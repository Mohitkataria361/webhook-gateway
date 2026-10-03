package worker

import (
	"context"
	"log"
	"sync"

	"webhook-gateway/internal/hub"
	"webhook-gateway/internal/queue"
	redisclient "webhook-gateway/internal/redis"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Pool manages a fixed-size pool of goroutine workers consuming from RabbitMQ.
type Pool struct {
	concurrency int
	mq          *queue.RabbitMQ
	dispatcher  *Dispatcher
	wg          sync.WaitGroup
}

// NewPool creates a worker pool wired with all required dependencies.
func NewPool(
	concurrency int,
	db *pgxpool.Pool,
	rc *redisclient.Client,
	mq *queue.RabbitMQ,
	h *hub.Hub,
) *Pool {
	return &Pool{
		concurrency: concurrency,
		mq:          mq,
		dispatcher:  NewDispatcher(db, rc, h),
	}
}

// Start launches N worker goroutines that each pull messages from RabbitMQ
// and process them via the Dispatcher. Blocks until ctx is cancelled.
func (p *Pool) Start(ctx context.Context) error {
	msgs, err := p.mq.Consume(p.concurrency)
	if err != nil {
		return err
	}

	log.Printf("[pool] Starting %d workers", p.concurrency)

	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.runWorker(ctx, i, msgs)
	}

	// Wait for ctx cancellation, then drain workers.
	<-ctx.Done()
	log.Println("[pool] Context cancelled — waiting for workers to finish")
	p.wg.Wait()
	log.Println("[pool] All workers stopped")
	return nil
}

// runWorker is the goroutine body: reads from the shared msgs channel,
// dispatches each job, and acks/nacks messages appropriately.
func (p *Pool) runWorker(ctx context.Context, id int, msgs <-chan amqp.Delivery) {
	defer p.wg.Done()
	log.Printf("[worker-%d] Started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[worker-%d] Shutting down", id)
			return

		case msg, ok := <-msgs:
			if !ok {
				log.Printf("[worker-%d] Message channel closed", id)
				return
			}

			job, err := queue.DecodeJob(msg.Body)
			if err != nil {
				log.Printf("[worker-%d] Failed to decode job: %v", id, err)
				msg.Nack(false, false) // Discard malformed messages
				continue
			}

			log.Printf("[worker-%d] Processing event %s (attempt %d)", id, job.EventID, job.AttemptNumber)

			if err := p.dispatcher.Dispatch(ctx, job, p.mq); err != nil {
				log.Printf("[worker-%d] Dispatch error for event %s: %v", id, job.EventID, err)
				msg.Nack(false, true) // Requeue on infrastructure errors
				continue
			}

			// Ack the message only after successful processing.
			msg.Ack(false)
		}
	}
}
