package controllers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/retawsolit/WeMeet-protocol/wemeet"
	"github.com/retawsolit/WeMeet-recorder/pkg/recorder"
	"github.com/retawsolit/WeMeet-recorder/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func (c *RecorderController) handleStopTask(req *wemeet.WeMeetToRecorder) bool {
	log.Infoln(fmt.Sprintf("received new stop task: %s, roomTableId: %d, roomId: %s, sId: %s", req.Task.String(), req.GetRoomTableId(), req.GetRoomId(), req.GetRoomSid()))

	var tasksToCheck []wemeet.RecordingTasks
	switch req.Task {
	case wemeet.RecordingTasks_STOP_RECORDING:
		tasksToCheck = append(tasksToCheck, wemeet.RecordingTasks_START_RECORDING)
	case wemeet.RecordingTasks_STOP_RTMP:
		tasksToCheck = append(tasksToCheck, wemeet.RecordingTasks_START_RTMP)
	case wemeet.RecordingTasks_STOP:
		// For a general STOP, try to stop both recording and RTMP.
		tasksToCheck = append(tasksToCheck, wemeet.RecordingTasks_START_RECORDING, wemeet.RecordingTasks_START_RTMP)
	}

	var found bool
	for _, task := range tasksToCheck {
		if process, ok := c.getAndDeleteRecorderInProgress(req.RoomTableId, task); ok && process != nil {
			// need to start the process in goroutine otherwise will be delay in reply,
			// and this will show error in the client
			go process.Close(nil)
			found = true
		}
	}

	return found
}

// getAndDeleteRecorderInProgress atomically loads and deletes a recorder from the sync.Map.
func (c *RecorderController) getAndDeleteRecorderInProgress(tableId int64, task wemeet.RecordingTasks) (*recorder.Recorder, bool) {
	id := fmt.Sprintf("%d-%d", tableId, task)
	val, ok := c.recordersInProgress.LoadAndDelete(id)
	if !ok {
		return nil, false
	}
	process, ok := val.(*recorder.Recorder)
	if !ok {
		log.Errorf("invalid type in recordersInProgress for id %s", id)
		return nil, false
	}
	return process, true
}

func (c *RecorderController) onAfterClose(req *wemeet.WeMeetToRecorder, filePath, fileName string, processErr error) {
	log.Infoln(fmt.Sprintf("onAfterClose called for task: %s, roomTableId: %d, roomId: %s, sId: %s", req.Task.String(), req.GetRoomTableId(), req.GetRoomId(), req.GetRoomSid()))

	// Atomically remove from map. This handles cleanup for crashes or other unexpected closures.
	// It's safe to call even if handleStopTask already removed it.
	c.recordersInProgress.Delete(fmt.Sprintf("%d-%d", req.RoomTableId, req.Task))

	// decrement process
	err := c.ns.UpdateCurrentProgress(false)
	if err != nil {
		log.Errorln(err)
	}

	// notify to wemeet
	toSend := &wemeet.RecorderToWeMeet{
		From:        "recorder",
		Status:      true,
		Msg:         "success",
		Task:        wemeet.RecordingTasks_END_RECORDING,
		RecordingId: req.RecordingId,
		RecorderId:  req.RecorderId,
		RoomTableId: req.RoomTableId,
	}
	if req.Task == wemeet.RecordingTasks_START_RTMP {
		toSend.Task = wemeet.RecordingTasks_END_RTMP
	}
	if processErr != nil {
		toSend.Status = false
		toSend.Msg = processErr.Error()
	}
	log.Infoln(fmt.Sprintf("notifyToWeMeet with data: %+v", toSend))

	_, err = utils.NotifyToWeMeet(c.cnf.WeMeetInfo.Host, c.cnf.WeMeetInfo.ApiKey, c.cnf.WeMeetInfo.ApiSecret, toSend, nil)
	if err != nil {
		log.Errorln(err)
	}

	if req.Task == wemeet.RecordingTasks_START_RECORDING {
		stat, err := os.Stat(path.Join(filePath, fileName))
		if err != nil {
			switch {
			case os.IsNotExist(err) && processErr != nil:
				// in this case, not found error is expected so, don't need to log
				// otherwise will create confusion
			default:
				log.Errorln(err)
			}
			return
		}
		if stat.Size() > 0 {
			go c.postProcessRecording(req, filePath, fileName)
		} else {
			log.Errorln("avoiding postProcessRecording of ", path.Join(filePath, fileName), "file because of 0 size")
		}
	}
}

func (c *RecorderController) postProcessRecording(req *wemeet.WeMeetToRecorder, filePath, currentFileName string) {
	finalFileName := fmt.Sprintf("%s.mp4", req.RecordingId)

	if c.cnf.Recorder.PostMp4Convert {
		var args []string
		args = append(args, strings.Split(c.cnf.FfmpegSettings.PostRecording.PreInput, " ")...)
		args = append(args, "-i", path.Join(filePath, currentFileName))
		args = append(args, strings.Split(c.cnf.FfmpegSettings.PostRecording.PostInput, " ")...)
		args = append(args, path.Join(filePath, finalFileName))
		log.Infoln("starting post recording ffmpeg process with args:", strings.Join(args, " "))

		_, err := exec.Command("ffmpeg", args...).CombinedOutput()
		if err != nil {
			log.Errorln(fmt.Sprintf("keeping the raw file: %s as output because of error from ffmpeg: %s", currentFileName, err.Error()))
			// remove the new file
			_ = os.Remove(path.Join(filePath, finalFileName))
			// keep the old file as output
			finalFileName = currentFileName
		} else {
			err = os.Remove(path.Join(filePath, currentFileName))
			if err != nil {
				log.Errorln(err)
			}
		}
	} else {
		// just rename
		err := os.Rename(path.Join(filePath, currentFileName), path.Join(filePath, finalFileName))
		if err != nil {
			log.Errorln(fmt.Sprintf("keeping the raw file: %s as output because of error during rename: %s", currentFileName, err.Error()))
			// keep the old file as output
			finalFileName = currentFileName
		}
	}

	outputFilePath := path.Join(filePath, finalFileName)
	stat, err := os.Stat(outputFilePath)
	if err != nil {
		log.Errorln(err)
		return
	}

	size := float32(stat.Size()) / 1000000.0
	var relativePath string

	// To robustly calculate the relative path, first ensure the base path is absolute.
	// This prevents errors when the configured path is relative (e.g., "./recordings").
	basePath, err := filepath.Abs(c.cnf.Recorder.CopyToPath.MainPath)
	if err != nil {
		log.WithError(err).Errorf("could not determine absolute path for main_path '%s', falling back to string trimming", c.cnf.Recorder.CopyToPath.MainPath)
		relativePath = strings.TrimPrefix(outputFilePath, c.cnf.Recorder.CopyToPath.MainPath)
	} else {
		// Now that we have an absolute base path, we can safely calculate the relative path.
		relativePath, err = filepath.Rel(basePath, outputFilePath)
		if err != nil {
			log.WithError(err).Errorf("could not make path relative for %s", outputFilePath)
			relativePath = strings.TrimPrefix(outputFilePath, basePath)
		}
	}

	toSend := &wemeet.RecorderToWeMeet{
		From:        "recorder",
		Status:      true,
		Task:        wemeet.RecordingTasks_RECORDING_PROCEEDED,
		Msg:         "success",
		RecordingId: req.RecordingId,
		RecorderId:  req.RecorderId,
		RoomTableId: req.RoomTableId,
		FilePath:    relativePath,
		FileSize:    float32(int(size*100)) / 100,
	}
	log.Infoln(fmt.Sprintf("notifyToWeMeet with data: %+v", toSend))

	_, err = utils.NotifyToWeMeet(c.cnf.WeMeetInfo.Host, c.cnf.WeMeetInfo.ApiKey, c.cnf.WeMeetInfo.ApiSecret, toSend, nil)
	if err != nil {
		log.Errorln(err)
	}

	// post-processing scripts
	if len(c.cnf.Recorder.PostProcessingScripts) == 0 {
		return
	}
	data := map[string]interface{}{
		"recording_id":  req.GetRecordingId(),
		"room_table_id": req.GetRoomTableId(),
		"room_id":       req.GetRoomId(),
		"room_sid":      req.GetRoomSid(),
		"file_name":     finalFileName,
		"file_path":     path.Join(filePath, finalFileName), // this will be the full path of the file
		"file_size":     size,
		"recorder_id":   req.GetRecorderId(),
	}
	marshal, err := json.Marshal(data)
	if err != nil {
		log.Errorln(err)
		return
	}

	for _, script := range c.cnf.Recorder.PostProcessingScripts {
		_, err := exec.Command("/bin/sh", script, string(marshal)).CombinedOutput()
		if err != nil {
			log.Errorln(err)
		}
	}
}
