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

package goethe

import (
	"fmt"
	"reflect"
)

// GetValues returns the reflection values for the arguments as specified by
// the method parameters.  Will fail if the wrong number of arguments is passed
// in or the arguments are not the correct type.  Otherwise will return
// the value versions of the arguments
func GetValues(method interface{}, args []interface{}) ([]reflect.Value, error) {
	typ := reflect.TypeOf(method)
	kin := typ.Kind()
	if kin != reflect.Func {
		return nil, fmt.Errorf("first argument of GetValues must be a function, it is %s", kin.String())
	}

	numIn := typ.NumIn()
	if numIn != len(args) {
		return nil, fmt.Errorf("Method has %d parameters, user passed in %d", numIn, len(args))
	}

	expectedTypes := make([]reflect.Type, numIn)
	for i := 0; i < numIn; i++ {
		expectedTypes[i] = typ.In(i)
	}

	arguments := make([]reflect.Value, numIn)
	for index, arg := range args {
		var argValue reflect.Value
		if arg == nil {
			argValue = reflect.New(expectedTypes[index]).Elem()
		} else {
			argValue = reflect.ValueOf(arg)
		}

		arguments[index] = argValue

		if !argValue.Type().AssignableTo(expectedTypes[index]) {
			return nil, fmt.Errorf("Value at index %d of type %s does not match method parameter of type %s",
				index, argValue.Type().String(), expectedTypes[index].String())

		}
	}

	return arguments, nil
}

// Invoke will call the method with the arguments, and ship any errors
// returned by the method to the errorQueue (which may be nil)
func Invoke(method interface{}, args []reflect.Value, errorQueue ErrorQueue) {
	val := reflect.ValueOf(method)
	retVals := val.Call(args)

	if errorQueue != nil {
		tid := GetGoethe().GetThreadID()

		// pick first returned error and return it
		for _, retVal := range retVals {
			if !retVal.IsNil() && retVal.CanInterface() {
				// First returned value that is not nill and is an error
				it := retVal.Type()
				isAnError := it.Implements(errorInterface)

				if isAnError {
					iFace := retVal.Interface()

					asErr := iFace.(error)

					errInfo := newErrorinformation(tid, asErr)

					errorQueue.Enqueue(errInfo)
				}
			}
		}
	}
}
