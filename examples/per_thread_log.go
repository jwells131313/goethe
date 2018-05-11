/*
 * DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
 *
 * Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved.
 *
 * The contents of this file are subject to the terms of either the GNU
 * General Public License Version 2 only ("GPL") or the Common Development
 * and Distribution License("CDDL") (collectively, the "License").  You
 * may not use this file except in compliance with the License.  You can
 * obtain a copy of the License at
 * https://glassfish.dev.java.net/public/CDDL+GPL_1_1.html
 * or packager/legal/LICENSE.txt.  See the License for the specific
 * language governing permissions and limitations under the License.
 *
 * When distributing the software, include this License Header Notice in each
 * file and include the License file at packager/legal/LICENSE.txt.
 *
 * GPL Classpath Exception:
 * Oracle designates this particular file as subject to the "Classpath"
 * exception as provided by Oracle in the GPL Version 2 section of the License
 * file that accompanied this code.
 *
 * Modifications:
 * If applicable, add the following below the License Header, with the fields
 * enclosed by brackets [] replaced by your own identifying information:
 * "Portions Copyright [year] [name of copyright owner]"
 *
 * Contributor(s):
 * If you wish your version of this file to be governed by only the CDDL or
 * only the GPL Version 2, indicate your decision by adding "[Contributor]
 * elects to include this software in this distribution under the [CDDL or GPL
 * Version 2] license."  If you don't indicate a single choice of license, a
 * recipient has the option to distribute your version of this file under
 * either the CDDL, the GPL Version 2 or to extend the choice of license to
 * its licensees as provided above.  However, if you add GPL Version 2 code
 * and therefore, elected the GPL Version 2 license, then the option applies
 * only if the new code is made subject to such option by the copyright
 * holder.
 */

package main

import (
	"bufio"
	"fmt"
	"github.com/jwells131313/goethe"
	"github.com/jwells131313/goethe/utilities"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// LocalLogger name of local log
	LocalLogger = "LogLocal"
)

type perThreadLogger struct {
	file   *os.File
	writer *bufio.Writer
}

var generator = rand.New(rand.NewSource(time.Now().Unix()))

func nextRandom() time.Duration {
	return time.Duration(generator.Int() % 10)
}

func getWriter() (*bufio.Writer, error) {
	ethe := utilities.GetGoethe()

	tl, err := ethe.GetThreadLocal(LocalLogger)
	if err != nil {
		return nil, err
	}

	raw, err := tl.Get()
	if err != nil {
		return nil, err
	}

	retVal, ok := raw.(*perThreadLogger)
	if !ok {
		return nil, fmt.Errorf("Unknown type for %v", raw)
	}

	return retVal.writer, nil
}

func sleeper(count *int32, cond *sync.Cond) error {
	fmt.Println("JRW(10) count=", *count)
	writer, err := getWriter()
	if err != nil {
		return err
	}

	val := atomic.AddInt32(count, 1)
	if val >= 100 {
		cond.Broadcast()
	}
	fmt.Println("JRW(10) val=", val)

	sleepTime := nextRandom()
	writer.WriteString(fmt.Sprintf("Will sleep for %ds\n", sleepTime))

	time.Sleep(sleepTime * time.Second)

	writer.WriteString("is it time to wake up already?!?\n")

	return nil
}

func initializeLogger(tl goethe.ThreadLocal) error {
	ethe := utilities.GetGoethe()
	tid := ethe.GetThreadID()

	fileName := fmt.Sprintf("log.%d", tid)

	f1, err := os.Create(fileName)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(f1)

	local := &perThreadLogger{
		file:   f1,
		writer: writer,
	}

	tl.Set(local)

	return nil
}

func destroyLogger(tl goethe.ThreadLocal) error {
	raw, err := tl.Get()
	if err != nil {
		return err
	}

	if raw == nil {
		return nil
	}

	ptl, ok := raw.(*perThreadLogger)
	if !ok {
		return fmt.Errorf("Unknown type for %v", ptl)
	}

	if ptl.writer != nil {
		ptl.writer.Flush()
	}

	if ptl.file != nil {
		err = ptl.file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func runSomeLoggingThreads() error {
	ch := make(chan bool)

	ethe := utilities.GetGoethe()

	fmt.Println("JRW(30) running runner")
	ethe.GoWithArgs(runner, ch)
	fmt.Println("JRW(40) after running runner")

	result := <-ch
	if !result {
		return fmt.Errorf("Failed in runner")
	}

	return nil
}

func runner(ch chan bool) error {
	fmt.Println("JRW(50) runner")
	ethe := utilities.GetGoethe()

	err := ethe.EstablishThreadLocal(LocalLogger, initializeLogger, destroyLogger)
	if err != nil {
		ch <- false
		return err
	}

	lock := ethe.NewGoetheLock()
	cond := sync.NewCond(lock)
	queue := ethe.NewBoundedFunctionQueue(1000)

	var count int32

	pool, err := ethe.NewPool("ThreadLocalExamplePool", 2, 10, 1*time.Second, queue, nil)
	if err != nil {
		ch <- false
		return err
	}

	pool.Start()

	for lcv := 0; lcv < 100; lcv++ {
		err = queue.Enqueue(sleeper, &count, cond)
		if err != nil {
			return err
		}
	}

	fmt.Println("JRW(60) before wait runner")

	lock.WriteLock()
	cond.Wait()
	lock.WriteUnlock()

	ch <- true

	return nil
}
