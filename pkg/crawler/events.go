package crawler

import (
        "sync"
        "time"
)

// Event is the kind of progress notification the crawler emits.
type Event int

const (
        EventStart Event = iota
        EventFetchStart
        EventFetchOK
        EventFetchError
        EventSave
        EventEnqueue
        EventSkipped
        EventEnd
        EventLog
        EventPause
        EventResume
)

// String returns a human-readable name for the event.
func (e Event) String() string {
        switch e {
        case EventStart:
                return "start"
        case EventFetchStart:
                return "fetch_start"
        case EventFetchOK:
                return "fetch_ok"
        case EventFetchError:
                return "fetch_error"
        case EventSave:
                return "save"
        case EventEnqueue:
                return "enqueue"
        case EventSkipped:
                return "skipped"
        case EventEnd:
                return "end"
        case EventLog:
                return "log"
        case EventPause:
                return "pause"
        case EventResume:
                return "resume"
        }
        return "unknown"
}

// ProgressEvent is the payload emitted by the crawler on every meaningful
// state change. GUI components subscribe to these events to update their
// views without polling the Stats counters.
//
// NOTE: We do not embed the atomic-containing Stats struct here because
// atomic.Int64 has a noCopy marker that prevents it from being passed by
// value. Instead we copy the counter values into the plain int64 fields
// below at emit time.
type ProgressEvent struct {
        Type  Event
        Time  time.Time
        URL   string
        Path  string // local mirror path, when applicable
        Bytes int64
        Msg   string // free-form message (for EventLog / EventFetchError)

        // Counter snapshot at the moment of the event
        Visited    int64
        Downloaded int64
        Failed     int64
        Skipped    int64
        TotalBytes int64
        Pages      int64
        Assets     int64
        StartTime  time.Time

        // Job metadata. When the crawler is run as part of a job queue (e.g.
        // from the GUI's "Start All Jobs" button), the orchestrator stamps
        // every event with the index of the current job (0-based), the total
        // number of jobs in the queue, and the seed URL of the current job.
        // This lets the UI render "Job 2 of 5: https://example.com" without
        // having to track job state itself. When not running in a queue,
        // JobIndex and JobTotal are 0.
        JobIndex int    `json:"job_index,omitempty"`
        JobTotal int    `json:"job_total,omitempty"`
        JobURL   string `json:"job_url,omitempty"`
}

// EventEmitter is a tiny pub/sub for ProgressEvents. Multiple subscribers
// can register; each gets its own buffered channel. Sending is non-blocking
// - if a subscriber's channel is full, the event is dropped to avoid
// blocking the crawler.
type EventEmitter struct {
        mu          sync.RWMutex
        subscribers map[chan ProgressEvent]struct{}
}

// NewEventEmitter constructs a fresh emitter with no subscribers.
func NewEventEmitter() *EventEmitter {
        return &EventEmitter{
                subscribers: make(map[chan ProgressEvent]struct{}),
        }
}

// Subscribe returns a channel that receives all subsequent events.
// The caller chooses the buffer size; events that don't fit are dropped.
// The returned channel should be drained by the caller; otherwise the
// emitter will silently drop events (it never blocks).
func (e *EventEmitter) Subscribe(buffer int) chan ProgressEvent {
        if buffer <= 0 {
                buffer = 32
        }
        ch := make(chan ProgressEvent, buffer)
        e.mu.Lock()
        e.subscribers[ch] = struct{}{}
        e.mu.Unlock()
        return ch
}

// Unsubscribe removes the channel from the emitter and closes it.
func (e *EventEmitter) Unsubscribe(ch chan ProgressEvent) {
        e.mu.Lock()
        if _, ok := e.subscribers[ch]; ok {
                delete(e.subscribers, ch)
                close(ch)
        }
        e.mu.Unlock()
}

// Emit sends the event to every subscriber. Non-blocking: if a
// subscriber's buffer is full, the event is dropped for that subscriber.
func (e *EventEmitter) Emit(ev ProgressEvent) {
        if e == nil {
                return
        }
        e.mu.RLock()
        defer e.mu.RUnlock()
        for ch := range e.subscribers {
                select {
                case ch <- ev:
                default:
                        // drop event to keep crawler responsive
                }
        }
}

// Close closes all subscriber channels. After Close is called, the emitter
// should not be reused.
func (e *EventEmitter) Close() {
        e.mu.Lock()
        defer e.mu.Unlock()
        for ch := range e.subscribers {
                close(ch)
                delete(e.subscribers, ch)
        }
}
