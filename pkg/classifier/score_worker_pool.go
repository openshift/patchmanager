package classifier

import (
	"github.com/enriquebris/goworkerpool"
	"github.com/mfojtik/patchmanager/pkg/github"
)

type ScoringPool interface {
	WithCallback(func(interface{})) ScoringPool
	WaitForFinish() error
	Add(...*github.PullRequest) error
}

type scoringPool struct {
	classifier Classifier
	pool       *goworkerpool.Pool
	callback   func(interface{})
}

var _ ScoringPool = &scoringPool{}

// NewScoringWorkerPool return a worker pool for given classifier that is able to score multiple pull request in parallel (with reasonable concurency).
// This speed up scoring pull requests as requests to bugzilla can be slow.
func NewScoringWorkerPool(classifier Classifier) ScoringPool {
	pool, err := goworkerpool.NewPoolWithOptions(goworkerpool.PoolOptions{
		TotalInitialWorkers:          3,
		MaxWorkers:                   6,
		MaxOperationsInQueue:         100,
		WaitUntilInitialWorkersAreUp: true,
		LogVerbose:                   true,
	})
	if err != nil {
		panic(err)
	}
	p := scoringPool{
		classifier: classifier,
		pool:       pool,
		callback:   func(i interface{}) {},
	}
	p.pool.SetWorkerFunc(func(pullInterface interface{}) bool {
		pull, ok := pullInterface.(*github.PullRequest)
		if !ok {
			return true
		}
		pull.Score = p.classifier.Score(pull)
		return true
	})
	return &p
}

func (p *scoringPool) WithCallback(fn func(interface{})) ScoringPool {
	p.callback = fn
	return p
}

func (p *scoringPool) WaitForFinish() error {
	return p.pool.Wait()
}

func (p *scoringPool) Add(pulls ...*github.PullRequest) error {
	go func() {
		for i := range pulls {
			p.pool.AddTaskCallback(pulls[i], p.callback)
		}
		p.pool.LateKillAllWorkers()
	}()
	return nil
}
