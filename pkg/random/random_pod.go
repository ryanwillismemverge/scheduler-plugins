package randompod

import (
	"context"
	"math/rand"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)


type RandomPod struct{
	handle framework.Handle
}

// Name is the name of the plugin used in the Registry and configurations.
const Name = "RandomPod"

var _ framework.ScorePlugin = &RandomPod{}

func (rp *RandomPod) Name() string {
	return Name
}

func (rp *RandomPod) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	score := rand.Int63n(100) + 1
	return score, framework.NewStatus(framework.Success)
}

func (rp *RandomPod) ScoreExtensions() framework.ScoreExtensions {
	return rp
}

func (rp *RandomPod) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	// If the scores are already in the range 1-100, there's no need to normalize them.
	// So, just return a successful status.
	return framework.NewStatus(framework.Success)
}

func New(_ runtime.Object, h framework.Handle) (framework.Plugin, error){
	return &RandomPod{handle: h}, nil
}