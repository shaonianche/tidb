// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/pkg/disttask/framework/proto"
	"github.com/pingcap/tidb/pkg/util/chunk"
	"github.com/pingcap/tidb/pkg/util/logutil"
	"go.uber.org/zap"
)

func row2TaskBasic(r chunk.Row) *proto.TaskBase {
	createTime, _ := r.GetTime(7).GoTime(time.Local)
	extraParams := proto.ExtraParams{}
	if !r.IsNull(10) {
		str := r.GetJSON(10).String()
		if err := json.Unmarshal([]byte(str), &extraParams); err != nil {
			logutil.BgLogger().Error("unmarshal task extra params", zap.Error(err))
		}
	}
	task := &proto.TaskBase{
		ID:           r.GetInt64(0),
		Key:          r.GetString(1),
		Type:         proto.TaskType(r.GetString(2)),
		State:        proto.TaskState(r.GetString(3)),
		Step:         proto.Step(r.GetInt64(4)),
		Priority:     int(r.GetInt64(5)),
		Concurrency:  int(r.GetInt64(6)),
		CreateTime:   createTime,
		TargetScope:  r.GetString(8),
		MaxNodeCount: int(r.GetInt64(9)),
		ExtraParams:  extraParams,
		Keyspace:     r.GetString(11),
	}
	return task
}

// Row2Task converts a row to a task.
func Row2Task(r chunk.Row) *proto.Task {
	taskBase := row2TaskBasic(r)
	task := &proto.Task{TaskBase: *taskBase}
	var startTime, updateTime time.Time
	if !r.IsNull(12) {
		startTime, _ = r.GetTime(12).GoTime(time.Local)
	}
	if !r.IsNull(13) {
		updateTime, _ = r.GetTime(13).GoTime(time.Local)
	}
	task.StartTime = startTime
	task.StateUpdateTime = updateTime
	task.Meta = r.GetBytes(14)
	task.SchedulerID = r.GetString(15)
	if !r.IsNull(16) {
		errBytes := r.GetBytes(16)
		stdErr := errors.Normalize("")
		err := stdErr.UnmarshalJSON(errBytes)
		if err != nil {
			logutil.BgLogger().Error("unmarshal task error", zap.Error(err))
			task.Error = errors.New(string(errBytes))
		} else {
			task.Error = stdErr
		}
	}
	if !r.IsNull(17) {
		str := r.GetJSON(17).String()
		if err := json.Unmarshal([]byte(str), &task.ModifyParam); err != nil {
			logutil.BgLogger().Error("unmarshal task modify param", zap.Error(err))
		}
	}
	return task
}

// row2BasicSubTask converts a row to a subtask with basic info
func row2BasicSubTask(r chunk.Row) *proto.SubtaskBase {
	taskIDStr := r.GetString(2)
	tid, err := strconv.Atoi(taskIDStr)
	if err != nil {
		logutil.BgLogger().Warn("unexpected subtask id", zap.String("subtask-id", taskIDStr))
	}
	createTime, _ := r.GetTime(7).GoTime(time.Local)
	var ordinal int
	if !r.IsNull(8) {
		ordinal = int(r.GetInt64(8))
	}

	// subtask defines start time as bigint, to ensure backward compatible,
	// we keep it that way, and we convert it here.
	var startTime time.Time
	if !r.IsNull(9) {
		ts := r.GetInt64(9)
		startTime = time.Unix(ts, 0)
	}

	subtask := &proto.SubtaskBase{
		ID:          r.GetInt64(0),
		Step:        proto.Step(r.GetInt64(1)),
		TaskID:      int64(tid),
		Type:        proto.Int2Type(int(r.GetInt64(3))),
		ExecID:      r.GetString(4),
		State:       proto.SubtaskState(r.GetString(5)),
		Concurrency: int(r.GetInt64(6)),
		CreateTime:  createTime,
		Ordinal:     ordinal,
		StartTime:   startTime,
	}
	return subtask
}

// Row2SubTask converts a row to a subtask.
func Row2SubTask(r chunk.Row) *proto.Subtask {
	subtask := &proto.Subtask{
		SubtaskBase: *row2BasicSubTask(r),
	}

	// subtask defines update time as bigint, to ensure backward compatible,
	// we keep it that way, and we convert it here.
	var updateTime time.Time
	if !r.IsNull(10) {
		ts := r.GetInt64(10)
		updateTime = time.Unix(ts, 0)
	}

	subtask.UpdateTime = updateTime
	subtask.Meta = r.GetBytes(11)
	subtask.Summary = r.GetJSON(12).String()
	return subtask
}
