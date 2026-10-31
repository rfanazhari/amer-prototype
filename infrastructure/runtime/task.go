package runtime

import (
	"context"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/query/taskquery"
	"github.com/rfanazhari/amer-prototype/application/usecase/taskusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/taskhandler"
	"github.com/rfanazhari/amer-prototype/domain/event"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/taskrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/clock"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/eventqueue"
	"gorm.io/gorm"
)

// assembleTask wires one Task application graph, its storage adapter and completion subscriber for HTTP delivery. A nil subscriber selects the logging demonstration subscriber.
func assembleTask(db *gorm.DB, logs *logmanager.Application, subscriber func(context.Context, event.TaskCompleted) error) (taskhandler.Handler, *eventqueue.Queue) {
	if subscriber == nil {
		subscriber = func(ctx context.Context, e event.TaskCompleted) error {
			txn := logs.StartConsumer("")
			defer txn.End()
			logmanager.InfoWithContext(txn.ToContext(ctx), "task completed", map[string]string{"taskId": e.TaskID, "completedAt": e.CompletedAt.Format(time.RFC3339Nano)})
			return nil
		}
	}
	queue := &eventqueue.Queue{Capacity: 100, Consumer: subscriber, OnError: func(err error) {
		txn := logs.Start("", "subscriber failed", logmanager.TxnTypeOther)
		txn.NoticeError(err)
		txn.End()
	}}

	store := taskrepository.Store{DB: db}
	timer := clock.System{}
	handler := taskhandler.Handler{Create: taskusecase.Create{Repository: store, Clock: timer}, Start: taskusecase.Start{Repository: store, Clock: timer}, Complete: taskusecase.Complete{Repository: store, Clock: timer, Queue: queue}, Reschedule: taskusecase.Reschedule{Repository: store, Clock: timer}, Detail: taskquery.Detail{Repository: store, Clock: timer}, LogError: func(err error) {
		txn := logs.Start("", "request error", logmanager.TxnTypeOther)
		txn.NoticeError(err)
		txn.End()
	}}
	return handler, queue
}
