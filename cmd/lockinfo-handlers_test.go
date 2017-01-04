/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"fmt"
	"testing"
	"time"
)

// TestListLocksInfo - Test for listLocksInfo.
func TestListLocksInfo(t *testing.T) {
	// Initialize globalNSMutex to validate listing of lock
	// instrumentation information.
	isDistXL := false
	initNSLock(isDistXL)

	// Acquire a few locks to populate lock instrumentation.
	// Take 10 read locks on bucket1/prefix1/obj1
	for i := 0; i < 10; i++ {
		readLk := globalNSMutex.NewNSLock("bucket1", "prefix1/obj1")
		readLk.RLock()
	}

	// Take write locks on bucket1/prefix/obj{11..19}
	for i := 0; i < 10; i++ {
		wrLk := globalNSMutex.NewNSLock("bucket1", fmt.Sprintf("prefix1/obj%d", 10+i))
		wrLk.Lock()
	}

	testCases := []struct {
		bucket   string
		prefix   string
		relTime  time.Duration
		numLocks int
	}{
		// Test 1 - Matches all the locks acquired above.
		{
			bucket:   "bucket1",
			prefix:   "prefix1",
			relTime:  time.Duration(0 * time.Second),
			numLocks: 20,
		},
		// Test 2 - Bucket doesn't match.
		{
			bucket:   "bucket",
			prefix:   "prefix1",
			relTime:  time.Duration(0 * time.Second),
			numLocks: 0,
		},
		// Test 3 - Prefix doesn't match.
		{
			bucket:   "bucket1",
			prefix:   "prefix11",
			relTime:  time.Duration(0 * time.Second),
			numLocks: 0,
		},
	}

	for i, test := range testCases {
		actual := listLocksInfo(test.bucket, test.prefix, test.relTime)
		if len(actual) != test.numLocks {
			t.Errorf("Test %d - Expected %d locks but observed %d locks",
				i+1, test.numLocks, len(actual))
		}
	}
}
