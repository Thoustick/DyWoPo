package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
)

// Структура задачи
type Job string

// Результат (можно расширить, если надо)
type Result string

// WorkerPool управляет воркерами
type WorkerPool struct {
	jobs    chan Job
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	workers map[int]context.CancelFunc
	nextID  int
	wg      sync.WaitGroup
}

// Создание нового пула
func NewWorkerPool(buffer int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		jobs:    make(chan Job, buffer),
		ctx:     ctx,
		cancel:  cancel,
		workers: make(map[int]context.CancelFunc),
	}
}

// Добавление воркера
func (wp *WorkerPool) AddWorker() {
	select {
	case <-wp.ctx.Done():
		fmt.Println("AddWorker skipped: pool is shutting down")
		return
	default:
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	id := wp.nextID
	wp.nextID++

	ctx, cancel := context.WithCancel(wp.ctx)
	wp.workers[id] = cancel

	wp.wg.Add(1)
	go wp.worker(ctx, id)
	fmt.Printf("Worker %d added\n", id)
}

// Удаление воркера
func (wp *WorkerPool) RemoveWorker() {
	select {
	case <-wp.ctx.Done():
		fmt.Println("RemoveWorker skipped: pool is shutting down")
		return
	default:
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	for id, cancel := range wp.workers {
		cancel()
		delete(wp.workers, id)
		fmt.Printf("Worker %d removed\n", id)
		break
	}
}

// Завершение всей работы
func (wp *WorkerPool) Shutdown() {
	wp.cancel()
	close(wp.jobs)
	wp.wg.Wait()
}

// Воркеры читают задачи из канала и обрабатывают
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}
			fmt.Printf("Worker %d processed job: %s\n", id, job)
			time.Sleep(300 * time.Millisecond) // имитация обработки
		}
	}
}

// Отправка задачи в пул
func (wp *WorkerPool) Submit(job Job) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Submit failed: channel closed")
		}
	}()

	select {
	case wp.jobs <- job:
	default:
		fmt.Println("Queue full: job dropped")
	}
}

func main() {
	wp := NewWorkerPool(10)

	// Добавим 2 воркера на старте
	wp.AddWorker()
	// wp.AddWorker()

	// Завершение по Ctrl+C
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
		fmt.Println("\nShutting down...")
		wp.Shutdown()
		os.Exit(0)
	}()

	// Пример: подаём задачи и добавляем/удаляем воркеров
	go func() {
		for i := 0; i < 20; i++ {
			wp.Submit(Job(fmt.Sprintf("Task-%d", i)))
			if i == 5 {
				wp.AddWorker()
			}
			if i == 15 {
				wp.RemoveWorker()
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	select {} // бесконечно
}
