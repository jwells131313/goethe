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

package tests

import (
	"github.com/jwells131313/goethe"
	"testing"
	"time"
)

func TestEverySecondForTenSeconds(t *testing.T) {
	ethe := goethe.GetGoethe()

	var count int

	timer, err := ethe.ScheduleWithFixedDelay(0, 1*time.Second, nil, hi, &count)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer timer.Cancel()

	time.Sleep(10 * time.Second)

	if count != 10 && count != 11 {
		t.Errorf("expected ten but got %d", count)
		return
	}
}

func TestAtFixedRate(t *testing.T) {
	ethe := goethe.GetGoethe()

	var count int

	// add and sleep adds and sleeps for 2, but that should not affect the every second rate
	timer, err := ethe.ScheduleAtFixedRate(0, 1*time.Second, nil, addAndSleep, &count)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer timer.Cancel()

	time.Sleep(10 * time.Second)

	if count != 10 && count != 11 {
		t.Errorf("expected ten but got %d", count)
		return
	}

}

func TestRunOnceAndCancel(t *testing.T) {
	ethe := goethe.GetGoethe()

	var count int

	// add and sleep adds and sleeps for 2, but that should not affect the every second rate
	timer, err := ethe.ScheduleWithFixedDelay(0, 1*time.Second, nil, runOnceAndCancel, &count)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	time.Sleep(3 * time.Second)

	if count != 1 {
		t.Errorf("expected one but got %d", count)
		return
	}

	if timer.IsRunning() {
		t.Errorf("timer did not get cancelled?")
		return
	}
}

func hi(addToMe *int) {
	*addToMe = *addToMe + 1
}

func addAndSleep(addToMe *int) {
	*addToMe = *addToMe + 1

	time.Sleep(2 * time.Second)
}

func runOnceAndCancel(addToMe *int) {
	*addToMe = *addToMe + 1

	tl, _ := goethe.GetGoethe().GetThreadLocal(goethe.TimerThreadLocal)
	if tl != nil {
		iface, _ := tl.Get()

		if iface != nil {
			timer := iface.(goethe.Timer)

			timer.Cancel()
		}
	}
}
