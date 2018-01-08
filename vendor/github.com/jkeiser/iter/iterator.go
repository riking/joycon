package iter

import "errors"

// Error returned from Iterator.Next() when iteration is done.
var FINISHED = errors.New("FINISHED")

// Iterator letting you specify just two functions defining iteration. All
// helper functions utilize this.
//
// Iterator is designed to avoid common pitfalls and to allow common Go patterns:
// - errors can be yielded all the way up the chain to the caller
// - safe iteration in background channels
// - iterators are nil-clean (you can send nil values through the iterator)
// - allows for cleanup when iteration ends early
type Iterator struct {
	// Get the next value (or error).
	// nil, iter.FINISHED indicates iteration is complete.
	Next func() (interface{}, error)
	// Close out resources in case iteration terminates early. Callers are *not*
	// required to call this if iteration completes cleanly (but they may).
	Close func()
}

// Use to create an iterator from a single closure.  retun nil to stop iteration.
//
//   iter.NewSimple(func() interface{} { return x*2 })
func NewSimple(f func() interface{}) Iterator {
	return Iterator{
		Next: func() (interface{}, error) {
			val := f()
			if val == nil {
				return nil, FINISHED
			} else {
				return f, nil
			}
		},
		Close: func() {},
	}
}

// Transform values in the input.
//   iterator.Map(func(item interface{}) interface{} { item.(int) * 2 })
func (iterator Iterator) Map(mapper func(interface{}) interface{}) Iterator {
	return Iterator{
		Next: func() (interface{}, error) {
			item, err := iterator.Next()
			if err != nil {
				return nil, err
			}
			return mapper(item), nil
		},
		Close: func() { iterator.Close() },
	}
}

// Filter values from the input.
//   iterator.Select(func(item interface{}) bool { item.(int) > 10 })
func (iterator Iterator) Select(selector func(interface{}) bool) Iterator {
	return Iterator{
		Next: func() (interface{}, error) {
			for {
				item, err := iterator.Next()
				if err != nil {
					return nil, err
				}
				if selector(item) {
					return item, nil
				}
			}
		},
		Close: func() { iterator.Close() },
	}
}

// Concatenate multiple iterators together in one.
//   d := iter.Concat(a, b, c)
func Concat(iterators ...Iterator) Iterator {
	i := 0
	return Iterator{
		Next: func() (interface{}, error) {
			if i >= len(iterators) {
				return nil, FINISHED
			}
			item, err := iterators[i].Next()
			if err == FINISHED {
				i++
				return nil, err
			} else if err != nil {
				i = len(iterators)
				return nil, err
			}
			return item, nil
		},
		Close: func() {
			// Close out remaining iterators
			for ; i < len(iterators); i++ {
				iterators[i].Close()
			}
		},
	}
}

// Iterate over all values, calling a user-defined function for each one.
//   iterator.Each(func(item interface{}) { fmt.Println(item) })
func (iterator Iterator) Each(processor func(interface{})) error {
	defer iterator.Close()
	item, err := iterator.Next()
	for err == nil {
		processor(item)
		item, err = iterator.Next()
	}
	return err
}

// Iterate over all values, calling a user-defined function for each one.
// If an error is returned from the function, iteration terminates early
// and the EachWithError function returns the error.
//   iterator.EachWithError(func(item interface{}) error { return http.Get(item.(string)) })
func (iterator Iterator) EachWithError(processor func(interface{}) error) error {
	defer iterator.Close()
	item, err := iterator.Next()
	for err == nil {
		err = processor(item)
		if err == nil {
			item, err = iterator.Next()
		}
	}
	return err
}

// Convert the iteration directly to a slice.
//   var list []interface{}
//   list, err := iterator.ToSlice()
func (iterator Iterator) ToSlice() (list []interface{}, err error) {
	err = iterator.Each(func(item interface{}) {
		list = append(list, item)
	})
	return list, err
}
