/*
Generic forward-only iterator that is safe and leak-free.

This package is intended to support forward-only iteration in a variety of use
cases while avoiding the normal errors and leaks that can happen with iterators
in Go. It provides mechanisms for map/select filtering, background iteration
through a goroutine, and error handling throughout.

The type of the iterator is interface{}, so it can store anything, at the cost
that you have to cast it back out when you use it. This package can be used as
is, or used as an example for creating your own forward-only iterators of more
specific types.

  sum := 0
  iterator.Each(func(item interface{}) {
    sum = sum + item.(int)
  })

Motivation

With the lack of generics and a builtin iterator pattern, iterators have been
the topic of much discussion in Go. Here are the discussions that inspired this:

http://ewencp.org/blog/golang-iterators/: Ewan Cheslack-Postava's
discussion of the major iteration patterns. Herein we have chosen the closure
pattern for iterator implementation, and given the choice of callback and
channel patterns for iteration callers.

http://blog.golang.org/pipelines: A March 2014 discussion of pipelines
on the go blog presents some of the pitfalls of channel iteration, suggesting
the "done" channel implementation to compensate.

Creating Iterators

Simple error- and cleanup-free iterators can be easily created:

  // Create a simple iterator from a function
  val := 1
  iterator := iter.NewSimple(func() interface{} {
    val = val * 2;
    return val > 2000 ? val : nil // nil terminates iteration
  })

Typically you will create iterators in packages ("please iterate over this
complicated thing").  You will often handle errors and have cleanup to do.
iter supports both of these. You can create a fully-functional iterator thusly:

  // Create a normal iterator parsing a file, close when done
  func ParseStream(reader io.ReadCloser) iter.Iterator {
    return iter.Iterator{
      Next: func() (iterator{}, error) {
        item, err := Parse()
        if item == nil && err == nil {
          return nil, iter.FINISHED
        }
        return item, err
      },
      Close: func() {
        reader.Close()
      }
    }
  }

Iterating

Callback iteration looks like this and handles any bookkeeping automatically:

  // Iterate over all values
  err := iterator.Each(func(item interface{}) {
    fmt.Println(item)
  })

Sometimes you need to handle errors:

  // Iterate over all values, terminating if processing has a problem
  var files []*File
  err := iterator.EachWithError(func(item interface{}) error {
    file, err = os.Open(item.(string))
    if err == nil {
      files = append(files, file)
    }
    return err
  })

Raw iteration looks like this:

  defer iterator.Close() // allow the iterator to clean itself up
  item, err := iterator.Next()
  for err == nil {
    ... // do stuff with value
    item, err = iterator.Next()
  }
  if err != iter.FINISHED {
    ... // handle error
  }

Background goroutine iteration (using channels) deserves special mention:

  // Produce the values in a goroutine, cleaning up safely afterwards.
  // This allows background iteration to continue at its own pace while we
  // perform blocking operations in the foreground.
  var responses []http.Response
  err := iterator.BackgroundEach(1000, func(item interface{}) error {
    response, err := http.Get(item.(string))
    if err == nil {
      responses = append(list, response)
    }
    return err
  })

Utilities

There are several useful functions provided to work with iterators:

  // Square the ints
  squaredIterator, err := iterator.Map(func(item interface{}) interface{} { item.int() * 2 })

  // Select non-nil values
  nonNilIterator, err := iterator.Select(func(item interface{}) bool) { item != nil })

  // Produce a list
  list, err := iterator.ToList()

*/
package iter
