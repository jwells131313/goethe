[//]: # " DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER. "
[//]: # "  "
[//]: # " Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved. "
[//]: # "  "
[//]: # " The contents of this file are subject to the terms of either the GNU "
[//]: # " General Public License Version 2 only (''GPL'') or the Common Development "
[//]: # " and Distribution License(''CDDL'') (collectively, the ''License'').  You "
[//]: # " may not use this file except in compliance with the License.  You can "
[//]: # " obtain a copy of the License at "
[//]: # " https://oss.oracle.com/licenses/CDDL+GPL-1.1 "
[//]: # " or LICENSE.txt.  See the License for the specific "
[//]: # " language governing permissions and limitations under the License. "
[//]: # "  "
[//]: # " When distributing the software, include this License Header Notice in each "
[//]: # " file and include the License file at LICENSE.txt. "
[//]: # "  "
[//]: # " GPL Classpath Exception: "
[//]: # " Oracle designates this particular file as subject to the ''Classpath'' "
[//]: # " exception as provided by Oracle in the GPL Version 2 section of the License "
[//]: # " file that accompanied this code. "
[//]: # "  "
[//]: # " Modifications: "
[//]: # " If applicable, add the following below the License Header, with the fields "
[//]: # " enclosed by brackets [] replaced by your own identifying information: "
[//]: # " ''Portions Copyright [year] [name of copyright owner]'' "
[//]: # "  "
[//]: # " Contributor(s): "
[//]: # " If you wish your version of this file to be governed by only the CDDL or "
[//]: # " only the GPL Version 2, indicate your decision by adding ''[Contributor] "
[//]: # " elects to include this software in this distribution under the [CDDL or GPL "
[//]: # " Version 2] license.''  If you don't indicate a single choice of license, a "
[//]: # " recipient has the option to distribute your version of this file under "
[//]: # " either the CDDL, the GPL Version 2 or to extend the choice of license to "
[//]: # " its licensees as provided above.  However, if you add GPL Version 2 code "
[//]: # " and therefore, elected the GPL Version 2 license, then the option applies "
[//]: # " only if the new code is made subject to such option by the copyright "
[//]: # " holder. "

# goethe

Threading utilities for GO

## Usage

This package provides several useful threading and locking utilities for use in golang.  In
particular goethe (pronounced ger-tay) threads have thread-ids which can be useful for logging
and in thread pools.

This package maintains one global goethe implementation which can be gotten using the
github.com/jwells131313/goethe/utilities.GetGoethe() method.

1. [ThreadID](#ThreadID)
2. [Recursive Locks](#recursive-locks)
3. [Thread Pools](#thread-pools)
4. [Thread Local Storage](#thread-local-storage)

For more information on the API please see the
[![GoDoc](https://godoc.org/github.com/jwells131313/goethe?status.svg)](https://godoc.org/github.com/jwells131313/goethe)

### ThreadID

Inside a goethe thread you can use the utilities.GetGoethe() method to get the implementation of Goethe and
use that to get the ID of the thread in which the code is currently running:

```go
package foo

import (
	"fmt"
	"github.com/jwells131313/goethe/utilities"
)

func basic() {
	goethe := utilities.GetGoethe()

	channel := make(chan int64)

	goethe.Go(func() {
		// A thread ID!
		tid := goethe.GetThreadID()

		// Tell momma about our thread-id
		channel <- tid
	})

	threadID := <-channel
	fmt.Println("the threadID in my thread was ", threadID)
}
```

You can also use a method that takes arguments with goethe.  For this
use the GoWithArgs method as in the following example:

```go
package foo

import (
	"fmt"
	"github.com/jwells131313/goethe/utilities"
)

func addMe(a, b, c int, ret chan int) {
	ret <- a + b + c
}

func basicWithArgs() {
	goethe := utilities.GetGoethe()

	channel := make(chan int)

	goethe.GoWithArgs(addMe, 1, 2, 3, channel)

	sum := <-channel
	fmt.Println("the sum in my thread was ", sum)
}
```

### Recursive Locks

In goethe threads you can have recursive reader/write mutexes which obey the following rules:

* Only one writer lock is allowed into the critical section
* When holding a writer lock another writer lock may be acquired (counting)
* When holding a writer lock you may acquire a reader lock.  The writer lock remains in effect
* Many reader locks can be held on multiple different threads
* When holding a reader lock another reader lock may be acquired on the same thread (counting)
* When holding a writer lock you may not acquire a reader lock.  Doing so leads too easily to deadlocks
* Once a writer asks for the lock no more readers will be able to enter, so writers can starve readers

The following is an example of a recursive write lock
```go
package main

import (
	"github.com/jwells131313/goethe/utilities"
	"github.com/jwells131313/goethe"
)

var goether goethe.Goethe = utilities.GetGoethe()
var lock goethe.Lock = goether.NewGoetheLock()

func writer1() {
	lock.WriteLock()
	defer lock.WriteUnlock()

	writer2()
}

func writer2() {
	lock.WriteLock()
	defer lock.WriteUnlock()
}

func main() {
	goether.Go(writer1)
}
```

If you try to call a Lock or Unlock method of a goethe lock while not inside a goethe thread it
will return an error.

### Thread Pools

Thread pools use goethe threads so that you can use thread-ids, thread-locals and recursive
locks in your threaded application.

In order to create a thread pool you specify the min threads in the the pool, the max threads
in the pool and the decay time, which is the time a thread will be idle before being released
(if the number of threads is greater than the min).  You also give the pool an implementation
of a FunctionQueue.  You can also get a default implementation of FunctionQueue from the goethe
service.  However any implementation of FunctionQueue will work, which allows you to use
priority queues or other queue implementations.  Once a queue is associated with a pool you use
the queue Enqueue API to give jobs to the thread pool.

You can also give the pool an ErrorQueue.  Any non-nil errors returned from the enqueued jobs will
be placed on the ErrorQueue.  It is up to the application to check and drain the ErrorQueue for
errors

UNDER CONSTRUCTION need an example

### Thread Local Storage

Goethe threads can take advantage of named thread local storage.  Thread local storage is first
established by giving it a name, an initializer function and a destroyer function.  Then
in your goethe threads you can call GetThreadLocal with the name of your thread local and get
the type returned by the initializer.  Each thread has its own copy of the type so two threads
will not interfere with each other.  When the thread goes away the destructor for all named
thread local storage associated with that thread will be called.

UNDER CONSTRUCTION need an example

### Under Construction

In the future it is intended for goethe to provide the following:

* Scheduled execution of things (on goethe threads)
* locks you can try
* locks you can give up on after some duration
* Threading Utilities Requests?

![](https://github.com/jwells131313/goethe/blob/master/images/goth.jpg "Go Thread Utilities")
