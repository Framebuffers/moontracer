package dispatch

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/framebuffers/moontracer/internal/guard"
)

type DirectMessage struct {
	ID         string
	Sender     string
	Target     string
	Content    string
	Components []discordgo.MessageComponent  // optional
	Embeds     []*discordgo.MessageEmbed     // optional
}

/*
	Flow:
		1. NewDispatcher(session, numberOfWorkers): creates the dispatcher with a discordgo session, with an empty stack, mutex+cond pair, and a quit channel.
		2. d.Start(): creates a series of dispatchers (goroutines) equal to the number of workers set.
			a) Each one of them calls d.work(id) -> calls d.pop().
			b) The stack is empty, so each worker hit d.cond.Wait() and sleep on the mutex.
		3. d.Push(msg): locks the mutex, pushes the message to the stack, unlocks mutex, cals d.cond.Signal().
			a) signal() wakes only one sleeping worker.
		4. d.Pop(): in the woken worker, reacquires the mutex released by Wait(), checks the stack is non-empty, pops last element, unlocks, returns the message.
		5. d.work(): a worker now has a DirectMessage. It calls d.send(msg).
		6. d.send(msg): two Discord API calls are made, in sequence:
			a) UserChannelCreate(target.ID): opens/reuses a DM channel with the target player.
			b) ChannelMessageSend(channel.ID, content): sends the actual text to the target player.
			c) Note: If either fails, the eror wraps the context (the step and ID) and returns.
				i. back in work(), this error is logged.
				ii. the worker loops back to pop() and sleeps waiting a new job.
		7. d.Stop(): Closes the quit channel, broadcasts to all sleeping workers that it is closing.
			a) each woken worker checks <-d.quit, sees if it's closed and pop() returns false.
			b) work() logs shutdown and goroutine exits.

	Notes:
		A stack+mutex prevents workers stepping on each other *and* keeps the order the messages were sent in (with the LIFO structure of a stack).
		The worker that finishes first, pops the next message. Because of this, no worker can pop the same message.
*/

// Dispatcher manages a stack of outbound DMs processed by a pool of workers.
type Dispatcher struct {
	session *discordgo.Session
	DryRun  bool // when true, log DMs instead of sending them

	mu    sync.Mutex
	stack []DirectMessage
	cond  *sync.Cond

	workers int
	quit    chan struct{}
}

func NewDispatcher(session *discordgo.Session, workers int) *Dispatcher {
	if workers < 1 {
		workers = 1
	}
	d := &Dispatcher{
		session: session,
		DryRun:  os.Getenv("SAFE_MODE") == "true",
		workers: workers,
		quit:    make(chan struct{}),
	}
	d.cond = sync.NewCond(&d.mu)
	return d
}

// Start launches all workers. Each worker pops from the stack and sends.
func (d *Dispatcher) Start() {
	for i := 1; i <= d.workers; i++ {
		go d.work(i)
	}
}

// Stop signals all workers to drain and exit.
func (d *Dispatcher) Stop() {
	close(d.quit)
	d.cond.Broadcast()
}

// Push adds a message to the top of the stack.
func (d *Dispatcher) Push(msg DirectMessage) {
	d.mu.Lock()
	d.stack = append(d.stack, msg)
	d.mu.Unlock()
	d.cond.Signal()
}

/*
Pending returns a snapshot of messages currently queued but not yet sent.

Pending is useful for diagnostics and for tests that need to inspect dispatch state
without starting worker goroutines.
*/
func (d *Dispatcher) Pending() []DirectMessage {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]DirectMessage, len(d.stack))
	copy(out, d.stack)
	return out
}

func (d *Dispatcher) pop() (DirectMessage, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for len(d.stack) == 0 {
		select {
		case <-d.quit:
			return DirectMessage{}, false
		default:
		}
		d.cond.Wait()
		select {
		case <-d.quit:
			return DirectMessage{}, false
		default:
		}
	}

	n := len(d.stack) - 1
	msg := d.stack[n]
	d.stack = d.stack[:n]
	return msg, true
}

func (d *Dispatcher) work(id int) {
	for {
		msg, ok := d.pop()
		if !ok {
			log.Printf("dispatcher: worker %d shutting down", id)
			return
		}
		if err := d.send(msg); err != nil {
			log.Printf("dispatcher: worker %d failed to send message %s: %s", id, msg.ID, err)
		}
	}
}

func (d *Dispatcher) send(msg DirectMessage) error {
	if d.DryRun && msg.Target != guard.DebugAdminID {
		log.Printf("dispatcher: [SAFE_MODE] would DM user %s (msg %s): %s", msg.Target, msg.ID, truncate(msg.Content, 80))
		return nil
	}
	if d.DryRun {
		log.Printf("dispatcher: [SAFE_MODE] sending DM to debug admin %s (whitelisted)", msg.Target)
	}

	channel, err := d.session.UserChannelCreate(msg.Target)
	if err != nil {
		return fmt.Errorf("dispatcher: create DM channel for %s: %w", msg.ID, err)
	}

	_, err = d.session.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Content:    msg.Content,
		Components: msg.Components,
		Embeds:     msg.Embeds,
	})
	if err != nil {
		return fmt.Errorf("dispatcher: send to channel %s: %w", channel.ID, err)
	}

	log.Printf("dispatcher: sent message %s to %s", msg.ID, msg.ID)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
