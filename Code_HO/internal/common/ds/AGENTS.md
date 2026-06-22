<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Data Structures (ds)

## Purpose
Generic, thread-safe data structures for StormSIM. Provides a type-safe FIFO queue with mutex protection and a task queue wrapper with event type validation for use in UE/gNB contexts and worker pool coordination.

## Key Files
| File | Description |
|------|-------------|
| `queue.go` | Generic thread-safe FIFO queue with `sync.RWMutex` protection |
| `task.go` | `Tasks[T]` wrapper combining Queue with `model.EventType` validation |

## Subdirectories
None.

## For AI Agents

### Generic Queue

Thread-safe FIFO queue using Go generics. All operations are mutex-protected.

```go
// Create a queue
q := &ds.Queue[int]{}

// Add items
q.Enqueue(1)
q.Enqueue(2)

// Remove and return first item
val, ok := q.Dequeue()  // val=1, ok=true

// Peek without removing
val, ok := q.Peek()     // val=2, ok=true

// Remove first item without returning
q.Remove()

// Check state
isEmpty := q.IsEmpty()  // false
len := q.Len()          // 1
```

**Thread Safety**: Uses `sync.RWMutex` with full Lock/Unlock for all operations. Safe for concurrent access from multiple goroutines.

### Tasks with Validation

Wrapper around Queue that validates tasks against a set of allowed `model.EventType` values. Invalid tasks are automatically removed from the queue.

```go
// Define valid event types for this task queue
validEvents := []model.EventType{
    model.RegistrationEvent,
    model.AuthEvent,
    model.PduSessionEvent,
}

// Create typed task queue
tasks := ds.NewTasks[MyTaskType](validEvents)

// Assign tasks to queue
tasks.AssignTask(someTask)

// Pop a task
task, ok := tasks.PopTask()

// Peek without removing
task, ok := tasks.PeekTask()

// Validate a task type (removes from queue if invalid)
eventType := model.RegistrationEvent
isValid := tasks.CheckValidTask(&eventType)  // true if in validEvents
```

**CheckValidTask Behavior**:
- Returns `true` if the event type is in the valid set
- Returns `false` AND dequeues the front item if the event type is NOT valid
- Used to filter out unexpected/invalid tasks during processing

### Usage Patterns

**UE Context Task Queue**:
```go
// In UE context initialization
ue.taskQueue = ds.NewTasks[*SomeTask]([]model.EventType{
    model.RegistrationEvent,
    model.DeregistrationEvent,
    model.PduSessionEvent,
})

// During event processing
if task, ok := ue.taskQueue.PopTask(); ok {
    if ue.taskQueue.CheckValidTask(&task.EventType) {
        // Process valid task
    }
}
```

**Producer-Consumer Pattern**:
```go
// Producer goroutine
tasks.AssignTask(newTask)

// Consumer goroutine
for {
    if task, ok := tasks.PopTask(); ok {
        process(task)
    }
    // Use channel or condition variable for blocking wait
}
```

### Concurrency Notes

- **Queue**: All methods acquire full lock (`Lock()`), not read lock. This is a minor inefficiency for read operations but ensures consistency.
- **Tasks**: Delegates to internal Queue, inherits thread safety.
- **No blocking**: Neither structure blocks on empty queue. Caller must handle empty case via `ok` return value.

## Dependencies

### Internal
- `pkg/model` - `EventType` for task validation

### External
- None (only standard library `sync`)

<!-- MANUAL: Add notes about specific data structure implementation details here -->
