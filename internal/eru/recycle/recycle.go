package recycle

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/projecteru2/core/log"
	virttypes "github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/eru/common"
	"github.com/projecteru2/yavirt/internal/eru/store"
	corestore "github.com/projecteru2/yavirt/internal/eru/store/core"
	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/service"
	"github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/utils/notify/bison"
	"github.com/samber/lo"
	"google.golang.org/grpc/status"
)

var (
	interval   = 1 * time.Minute
	deleteWait = 15 * time.Second
	sto        store.Store
)

func fetchWorkloads() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wrks, err := sto.ListNodeWorkloads(ctx, configs.Hostname())
	if err != nil {
		return nil, err
	}
	ids := lo.Map(wrks, func(w *types.Workload, _ int) string {
		return w.ID
	})
	return ids, nil
}

func deleteGuest(svc service.Service, eruID string) error {
	logger := log.WithFunc("deleteGuest")
	// when core delete a workload, it will delete the record in etcd first and then delete the workload
	// so there is a time window in which the guest is a dangling guest, so we wait for a while and wait the deletion finished
	// TODO better way to detect if a guest is in deletion
	time.Sleep(deleteWait)

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	logger.Infof(ctx, "[recycle] start to remove dangling guest %s", eruID)
	// since guest deletion is a dangerous operation here,
	// so we check eru again
	wrk, err := sto.GetWorkload(ctx, eruID)
	logger.Infof(ctx, "[recycle]  guest %s, wrk: %v, err: %v", eruID, wrk, err)
	if err == nil {
		logger.Errorf(ctx, err, "[recycle] BUG: dangling guest %s is still in eru", eruID)
		return errors.Errorf("BUG: dangling guest %s is still in eru", eruID)
	}

	e, ok := status.FromError(err)
	if !ok {
		return err
	}
	if e.Code() == 1051 && strings.Contains(e.Message(), "entity count invalid") { //nolint
		logger.Infof(ctx, "[recycle] start to remove local guest %s", eruID)
		// When creating a guest, the core first creates the workload and then creates a record in ETCD.
		// Therefore, within the time window between these two operations, we may incorrectly detect dangling guests.
		// To prevent this situation, we create a creation session locker when creating a guest and check this locker here.
		flck := utils.NewCreateSessionFlock(utils.VirtID(eruID))
		if flck.FileExists() {
			// creation session locker file exists
			// it means this guest is in creation
			logger.Warnf(ctx, "[recycle] guest %s in creation", eruID)
			return fmt.Errorf("guest %s is in creation", eruID)
		}

		if err := svc.ControlGuest(ctx, utils.VirtID(eruID), virttypes.OpDestroy, true); err != nil {
			logger.Errorf(ctx, err, "[recycle] failed to remove dangling guest %s", eruID)
			return err
		}
		notifier := bison.GetService()
		log.Debugf(ctx, "[recycle] notifier: %v", notifier)
		if notifier != nil {
			msgList := []string{
				"<font color=#00CC33 size=10>delete dangling successfully </font>",
				"---",
				"",
				fmt.Sprintf("- **node:** %s", configs.Hostname()),
				fmt.Sprintf("- **id:** %s", eruID),
			}
			text := "\n" + strings.Join(msgList, "\n")
			if err := notifier.SendMarkdown(context.TODO(), "delete dangling guest", text); err != nil {
				logger.Warnf(ctx, "[recycle] failed to send dingtalk message: %v", err)
			}
		}
		return nil
	}

	return err
}

func startLoop(ctx context.Context, svc service.Service) {
	log.WithFunc("startLoop").Infof(ctx, "[recycle] starting recycle loop")
	defer log.WithFunc("startLoop").Infof(ctx, "[recycle] recycle loop stopped")

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}

		coreIDs, err := fetchWorkloads()
		if err != nil {
			continue
		}
		localIDs, err := svc.GetGuestIDList(context.Background())
		if err != nil {
			continue
		}
		coreMap := map[string]struct{}{}
		for _, id := range coreIDs {
			coreMap[id] = struct{}{}
		}
		for _, id := range localIDs {
			eruID := virttypes.EruID(id)
			if _, ok := coreMap[eruID]; ok {
				continue
			}
			go deleteGuest(svc, eruID) //nolint
		}
	}
}

func Setup(ctx context.Context, cfg *configs.Config, t *testing.T) (err error) {
	if t == nil {
		corestore.Init(ctx, &cfg.Eru)
		if sto = corestore.Get(); sto == nil {
			return common.ErrGetStoreFailed
		}
	} else {
		sto = storemocks.NewFakeStore()
	}
	return nil
}

func Run(ctx context.Context, svc service.Service) {
	go startLoop(ctx, svc)
}
