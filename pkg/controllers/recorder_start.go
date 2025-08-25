package controllers

import (
	"errors"
	"fmt"

	"github.com/retawsolit/WeMeet-protocol/wemeet"
	"github.com/retawsolit/WeMeet-recorder/pkg/recorder"
	"github.com/retawsolit/WeMeet-recorder/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func (c *RecorderController) handleStartTask(req *wemeet.WeMeetToRecorder) error {
	id := fmt.Sprintf("%d-%d", req.RoomTableId, req.Task)
	_, ok := c.recordersInProgress.Load(id)
	if ok {
		return errors.New("this request in progress")
	}
	log.Infoln(fmt.Sprintf("received new start task: %s, roomTableId: %d, roomId: %s, sId: %s", req.Task.String(), req.GetRoomTableId(), req.GetRoomId(), req.GetRoomSid()))

	rc := &recorder.Recorder{
		AppCnf:               c.cnf,
		Req:                  req,
		OnAfterStartCallback: c.onAfterStart,
		OnAfterCloseCallback: c.onAfterClose,
	}

	r := recorder.New(rc)
	// add in the list, otherwise if OnAfterCloseCallback called for an error,
	// then processes won't clean properly because of missing process record
	c.recordersInProgress.Store(id, r)

	var err error
	defer func() {
		if err != nil {
			log.Errorln(err)
			r.Close(err)
		}
	}()

	// start the process
	if err = r.Start(); err != nil {
		return err
	}

	if err = c.ns.UpdateCurrentProgress(true); err != nil {
		return err
	}

	return nil
}

func (c *RecorderController) onAfterStart(req *wemeet.WeMeetToRecorder) {
	log.Infoln(fmt.Sprintf("onAfterStart called for task: %s, roomTableId: %d, roomId: %s, sId: %s", req.Task.String(), req.GetRoomTableId(), req.GetRoomId(), req.GetRoomSid()))

	// notify to wemeet
	toSend := &wemeet.RecorderToWeMeet{
		From:        "recorder",
		Status:      true,
		Task:        req.Task,
		Msg:         "success",
		RecordingId: req.RecordingId,
		RecorderId:  req.RecorderId,
		RoomTableId: req.RoomTableId,
	}
	log.Infoln(fmt.Sprintf("notifyToWeMeet with data: %+v", toSend))

	_, err := utils.NotifyToWeMeet(c.cnf.WeMeetInfo.Host, c.cnf.WeMeetInfo.ApiKey, c.cnf.WeMeetInfo.ApiSecret, toSend, nil)
	if err != nil {
		log.Errorln(err)
	}
}
