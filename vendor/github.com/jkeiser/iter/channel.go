package iter

import "log"

// A result from a iterator.Go() channel.
type ChannelItem struct {
	Value interface{}
	Error error
}

// Runs the iterator in a goroutine, sending to a channel.
//
// The returned items channel will send ChannelItems to you, which can indicate
// either a value or an error.  If item.Error is set, iteration is ready
// to terminate.  Otherwise item.Value is the next value of the iteration.
//
// When you are done iterating, you must close the done channel. Even if iteration
// was terminated early, this will ensure that the goroutine and channel are
// properly cleaned up.
//
//   channel, done := iter.Go(iterator)
//   defer close(done) // if early return or panic happens, this will clean up the goroutine
//   for item := range channel {
//     if item.Error != nil {
//       // Iteration failed; handle the error and exit the loop
//       ...
//     }
//     value := item.Value
//     ...
//   }
func (iterator Iterator) Go() (items <-chan ChannelItem, done chan<- bool) {
	itemsChannel := make(chan ChannelItem)
	doneChannel := make(chan bool)
	go iterator.IterateToChannel(itemsChannel, doneChannel)
	return itemsChannel, doneChannel
}

// Iterates all items and sends them to the given channel.  Runs on the current
// goroutine (call go iterator.IterateToChannel to set it up on a new goroutine).
// This will close the items channel when done.  If the done channel is closed,
// iteration will terminate.
func (iterator Iterator) IterateToChannel(items chan<- ChannelItem, done <-chan bool) {
	defer close(items)
	err := iterator.EachWithError(func(result interface{}) error {
		select {
		case items <- ChannelItem{Value: result}:
			return nil
		case _, _ = <-done:
			// If we are told we're done early, we finish quietly.
			return FINISHED
		}
	})
	if err != nil {
		items <- ChannelItem{Error: err}
	}
}

// Iterate over the channels from a Go(), calling a user-defined function for each value.
// This function handles all anomalous conditions including errors, early
// termination and safe cleanup of the goroutine and channels.
func EachFromChannel(items <-chan ChannelItem, done chan<- bool, processor func(interface{}) error) error {
	defer close(done) // if early return or panic happens, this will clean up the goroutine
	for item := range items {
		if item.Error != nil {
			return item.Error
		}
		err := processor(item.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// Perform the iteration in the background concurrently with the Each() statement.
// Useful when the iterator or iteratee will be doing blocking work.
//
// The bufferSize parameter lets you set how far ahead the background goroutine can
// get.
//
//   iterator.BackgroundEach(100, func(item interface{}) { ... })
func (iterator Iterator) BackgroundEach(bufferSize int, processor func(interface{}) error) error {
	itemsChannel := make(chan ChannelItem, bufferSize)
	doneChannel := make(chan bool)
	go iterator.IterateToChannel(itemsChannel, doneChannel)
	return EachFromChannel(itemsChannel, doneChannel, processor)
}

// Iterate to a channel in the background.
//
//   for value := range iter.GoSimple(iterator) {
//     ...
//   }
//
// With this method, two undesirable things can happen:
// - if the iteration stops early due to an error, you will not be able to handle
//   it (the goroutine will log and panic, and the program will exit).
// - if callers panic or exit early without retrieving all values from the channel,
//   the goroutine is blocked forever and leaks.
//
// The Go() routine allows you to handle both of these issues, at a small cost to
// caller complexity.  BackgroundEach() provides a simple way to use Go(), as
// well.
//
// That said, if you can make guarantees about no panics or don't care, this
// method can make calling code easier to read.
func (iterator Iterator) GoSimple() (values <-chan interface{}) {
	mainChannel := make(chan interface{})
	go iterator.IterateToChannelSimple(mainChannel)
	return mainChannel
}

// Iterates all items and sends them to the given channel.  Runs on the current
// goroutine (call go iterator.IterateToChannelSimple() to set it up on a new goroutine).
// This will close the values channel when done.  See warnings about GoSimple()
// vs. Go() in the GoSimple() method.
func (iterator Iterator) IterateToChannelSimple(values chan<- interface{}) {
	defer close(values)
	err := iterator.Each(func(item interface{}) {
		values <- item
	})
	if err != nil {
		log.Fatalf("Iterator returned an error in GoSimple: %v", err)
	}
}
