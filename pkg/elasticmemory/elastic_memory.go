package elasticmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"bytes"
)

type ElasticMemory struct{
	handle framework.Handle
}

type ScoreResponse struct {
	Result map[string]int64 `json:"result"`
}

type PodData struct {
    Name           string `json:"name"`
    MemoryRequests string `json:"memory_requests"`
    MemoryLimits   string `json:"memory_limits"`
	AssignedNode   string `json:"assigned_node"`
}


const Name = "ElasticMemory"

var _ framework.ScorePlugin = &ElasticMemory{}
var _ framework.ReservePlugin = &ElasticMemory{}

func (em *ElasticMemory) Name() string {
	return Name
}

func (em *ElasticMemory) getScores() (map[string]int64, error) {
	response, err := http.Get("http://10.0.0.46:6789/m8s/scores")
	if err != nil {
		return nil, fmt.Errorf("Failed to get scores: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: %v", err)
	}

	var data ScoreResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("Failed to parse scores: %v", err)
	}

	return data.Result, nil
}

func (em *ElasticMemory) sendPodData(podDataList []PodData) error {
    jsonData, err := json.Marshal(podDataList)
    if err != nil {
        return fmt.Errorf("unable to marshal JSON: %w", err)
    }

    resp, err := http.Post("http://10.0.0.46:6789/m8s/pods", "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("unable to make HTTP request: %w", err)
    }
    defer resp.Body.Close()

    return nil
}

func (em *ElasticMemory) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	scores, err := em.getScores()
	if err != nil {
		return 0, framework.NewStatus(framework.Error, err.Error())
	}

	score, exists := scores[nodeName]
	if !exists {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("No score found for node %s", nodeName))
	}

	return score, framework.NewStatus(framework.Success)
}

func (em *ElasticMemory) Reserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
    var podDataList []PodData

    for i, container := range pod.Spec.Containers {
        memoryRequest := container.Resources.Requests[v1.ResourceMemory]
        memoryLimit := container.Resources.Limits[v1.ResourceMemory]
        fmt.Printf("Container %d requests %s memory\n", i, memoryRequest.String())

        podDataList = append(podDataList, PodData{
            Name:           pod.Name,
            MemoryRequests: memoryRequest.String(),
            MemoryLimits:   memoryLimit.String(),
			AssignedNode:   nodeName,
        })
    }

    err := em.sendPodData(podDataList)
    if err != nil {
        return framework.NewStatus(framework.Error, "Unable to send Pod data: "+err.Error())
    }

    return framework.NewStatus(framework.Success)
}

func (em *ElasticMemory) Unreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	// TODO: De-Alloc any CXL alloc'd memory
}


func (em *ElasticMemory) ScoreExtensions() framework.ScoreExtensions {
	return em
}

func (em *ElasticMemory) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	if len(scores) == 0 {
		return framework.NewStatus(framework.Error, "no scores available to normalize")
	}

	var maxScore int64
	for _, score := range scores {
		if score.Score > maxScore {
			maxScore = score.Score
		}
	}

	if maxScore == 0 {
		return framework.NewStatus(framework.Success)
	}

	for i := range scores {
		scores[i].Score = scores[i].Score * framework.MaxNodeScore / maxScore
	}

	return framework.NewStatus(framework.Success)
}

func New(_ runtime.Object, h framework.Handle) (framework.Plugin, error){
	return &ElasticMemory{handle: h}, nil
}
