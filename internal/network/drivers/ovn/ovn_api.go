package ovn

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cockroachdb/errors"
	"github.com/go-logr/logr"
	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/ovn-org/libovsdb/ovsdb"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/samber/lo"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

func normalizeAddr(addr string) string {
	if !strings.HasPrefix(addr, "tcp:") && !strings.HasPrefix(addr, "unix:") {
		if strings.HasPrefix(addr, "/") {
			addr = "unix:" + addr
		} else {
			addr = "tcp:" + addr
		}
	}
	return addr
}

func getCli(addrs []string, dbModelReq *model.ClientDBModel, monitor bool) (client.Client, error) {
	opts := make([]client.Option, 0, len(addrs)+3)
	for _, addr := range addrs {
		addr = normalizeAddr(addr)
		opts = append(opts, client.WithEndpoint(addr))
	}

	zlogger := log.GetGlobalLogger()
	slogHandler := slogzerolog.Option{Level: slog.LevelDebug, Logger: zlogger}.NewZerologHandler()
	ovsLogger := logr.FromSlogHandler(slogHandler)
	opts = append(opts, []client.Option{
		client.WithReconnect(15*time.Second, backoff.NewExponentialBackOff()),
		// client.WithInactivityCheck(1*time.Minute, 10*time.Second, backoff.NewExponentialBackOff()),
		client.WithLogger(&ovsLogger),
	}...)
	cli, err := client.NewOVSDBClient(*dbModelReq, opts...)
	if err != nil {
		return nil, err
	}
	if err = cli.Connect(context.TODO()); err != nil {
		return nil, err
	}
	if monitor {
		_, err = cli.MonitorAll(context.TODO())
	}
	return cli, err
}

func (d *Driver) getNBCli() (cli client.Client, err error) {
	if d.nbCli == nil {
		d.nbClientDBModel, _ = model.NewClientDBModel("OVN_Northbound", map[string]model.Model{
			"Logical_Switch":      &LogicalSwitch{},
			"Logical_Switch_Port": &LogicalSwitchPort{},
		})
		// dbModelReq.SetIndexes(map[string][]model.ClientIndex{
		// 	"Logical_Switch": {
		// 		{
		// 			Columns: []model.ColumnKey{
		// 				{
		// 					Column: "name",
		// 				},
		// 			},
		// 		},
		// 	},
		// })
		d.nbCli, err = getCli(d.cfg.NBAddrs, &d.nbClientDBModel, false)
	}
	return d.nbCli, err
}

func (d *Driver) getOVSDBCli() (cli client.Client, err error) {
	if d.ovsCli == nil {
		d.ovsClientDBModel, _ = model.NewClientDBModel("Open_vSwitch", map[string]model.Model{
			"Interface": &Interface{},
		})
		d.ovsCli, err = getCli([]string{d.cfg.OVSDBAddr}, &d.ovsClientDBModel, true)
	}
	return d.ovsCli, err
}

func (d *Driver) setExternalID(ifaceName, key, value string) error {
	iface := &Interface{
		Name: ifaceName,
	}
	cli, err := d.getOVSDBCli()
	if err != nil {
		return err
	}

	err = cli.Get(context.TODO(), iface)
	if err != nil {
		return errors.Wrapf(err, "failed to get interface %s", ifaceName)
	}
	iface.ExternalIDs[key] = value

	ops, err := cli.Where(iface).Update(iface)
	if err != nil {
		return errors.Wrapf(err, "failed to set external_id %s", key)
	}
	_, err = cli.Transact(context.TODO(), ops...)
	if err != nil {
		return errors.Wrapf(err, "failed to set external_id %s", key)
	}
	return nil
}

func (d *Driver) createLogicalSwitch(name string, subnet string) (string, error) { //nolint:unused
	cli, err := d.getNBCli()
	if err != nil {
		return "", err
	}
	obj := &LogicalSwitch{
		Name: name,
	}
	if subnet != "" {
		obj.Config = map[string]string{
			"subnet": subnet,
		}
	}
	ops, err := cli.Create(obj)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create logical switch %s", name)
	}
	opsRes, err := cli.Transact(context.TODO(), ops...)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create logical switch %s", name)
	}
	var ansUUID string
	if len(opsRes) > 0 {
		ansUUID = opsRes[0].UUID.GoUUID
	}
	return ansUUID, nil
}

func (d *Driver) getLogicalSwitch(uuid string) (*LogicalSwitch, error) {
	cli, err := d.getNBCli()
	if err != nil {
		return nil, err
	}
	obj := &LogicalSwitch{
		UUID: uuid,
	}
	v, err := selectHelper(cli, "Logical_Switch", &d.nbClientDBModel, obj)
	if err != nil {
		return nil, err
	}
	if ans, ok := v.(*LogicalSwitch); !ok {
		return nil, errors.New("failed to convert to LogicalSwitch")
	} else { //nolint
		return ans, nil
	}
}

func (d *Driver) getLogicalSwitchByName(name string) ([]*LogicalSwitch, error) {
	cli, err := d.getNBCli()
	if err != nil {
		return nil, err
	}
	obj := &LogicalSwitch{
		Name: name,
	}
	ans, err := selectHelper(cli, "Logical_Switch", &d.nbClientDBModel, obj)
	if err != nil {
		return nil, err
	}
	if ans == nil {
		return nil, nil
	}
	switch v := ans.(type) {
	case *LogicalSwitch:
		return []*LogicalSwitch{v}, nil
	case []*LogicalSwitch:
		return v, nil
	default:
		return nil, errors.New("failed to convert to LogicalSwitch")
	}
}

func (d *Driver) getOneLogicalSwitchByName(name string) (*LogicalSwitch, error) {
	lsList, err := d.getLogicalSwitchByName(name)
	if err != nil {
		return nil, err
	}
	switch len(lsList) {
	case 1:
		return lsList[0], nil
	case 0:
		return nil, fmt.Errorf("logical switch %s not found", name)
	default:
		return nil, fmt.Errorf("multiple logical switch %s found", name)
	}
}

func (d *Driver) deleteLogicalSwitch(name string) error { //nolint:unused
	cli, err := d.getNBCli()
	if err != nil {
		return err
	}
	ops, err := cli.Where(&LogicalSwitch{
		Name: name,
	}).Delete()
	if err != nil {
		return errors.Wrapf(err, "failed to delete logical switch %s", name)
	}
	_, err = cli.Transact(context.TODO(), ops...)
	return errors.Wrapf(err, "failed to delete logical switch %s", name)
}

func (d *Driver) createLogicalSwitchPort(args *types.EndpointArgs) (string, error) {
	cli, err := d.getNBCli()
	if err != nil {
		return "", err
	}
	namedUUID, err := newRowUUID()
	if err != nil {
		return "", err
	}
	obj := &LogicalSwitchPort{
		UUID: namedUUID,
		Name: LSPName(args.GuestID),
		// Type: "internal",
		Addresses: []string{
			fmt.Sprintf("%s dynamic", args.MAC),
		},
		PortSecurity: []string{
			args.MAC,
		},
	}
	ops, err := cli.Create(obj)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create logical switch port %s", LSPName(args.GuestID))
	}

	ls := &LogicalSwitch{
		UUID: args.OVN.LogicalSwitchUUID,
	}
	if ls.UUID == "" {
		ls, err = d.getOneLogicalSwitchByName(args.OVN.LogicalSwitchName)
		if err != nil {
			return "", err
		}
	}
	lsOps, err := cli.Where(ls).Mutate(ls, model.Mutation{
		Field:   &ls.Ports,
		Mutator: ovsdb.MutateOperationInsert,
		Value:   []string{namedUUID},
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to create logical switch port %s", LSPName(args.GuestID))
	}
	ops = append(ops, lsOps...)

	opsRes, err := cli.Transact(context.TODO(), ops...)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create logical switch port %s", LSPName(args.GuestID))
	}
	err = lo.Reduce(opsRes, func(r error, op ovsdb.OperationResult, _ int) error {
		if op.Error != "" {
			return errors.CombineErrors(r, errors.Newf("%s: %s", op.Error, op.Details))
		}
		return r
	}, nil)
	if err != nil {
		return "", err
	}
	var ansUUID string
	if len(opsRes) > 0 {
		ansUUID = opsRes[0].UUID.GoUUID
	}
	return ansUUID, nil
}

func (d *Driver) deleteLogicalSwitchPort(args *types.EndpointArgs) error {
	cli, err := d.getNBCli()
	if err != nil {
		return err
	}
	lsUUID := args.OVN.LogicalSwitchUUID
	if lsUUID == "" {
		ls, err := d.getOneLogicalSwitchByName(args.OVN.LogicalSwitchName)
		if err != nil {
			return err
		}
		lsUUID = ls.UUID
	}
	ls := &LogicalSwitch{
		UUID: lsUUID,
	}
	lsp, err := d.getLogicalSwitchPortByName(LSPName(args.GuestID))
	if err != nil {
		return err
	}
	uuid := lsp.UUID
	ops, err := cli.Where(ls).Mutate(ls, model.Mutation{
		Field:   &ls.Ports,
		Mutator: ovsdb.MutateOperationDelete,
		Value:   []string{uuid},
	})
	if err != nil {
		return err
	}
	_, err = cli.Transact(context.TODO(), ops...)
	if err != nil {
		return err
	}
	return nil
}

func selectHelper(cli client.Client, table string, cModel *model.ClientDBModel, obj any) (any, error) {
	ops, err := cli.Select(obj)
	if err != nil {
		return nil, err
	}
	opsRes, err := cli.Transact(context.TODO(), ops...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to run transaction for selection")
	}
	if opsRes[0].Error != "" {
		return nil, errors.Newf("%s:%s", opsRes[0].Error, opsRes[0].Details)
	}
	if len(opsRes[0].Rows) == 0 {
		return nil, nil //nolint:nilnil
	}
	dbModel, errList := model.NewDatabaseModel(cli.Schema(), *cModel)
	err = lo.Reduce(errList, func(r error, e error, _ int) error {
		return errors.CombineErrors(r, e)
	}, nil)
	if err != nil {
		return nil, err
	}

	ans := make([]any, 0, len(opsRes[0].Rows))
	for idx := range opsRes[0].Rows {
		row := opsRes[0].Rows[idx]
		var uuid string
		if v, ok := row["_uuid"]; ok {
			uuid = v.(ovsdb.UUID).GoUUID
		}
		mod, err := model.CreateModel(dbModel, table, &row, uuid)
		if err != nil {
			return nil, err
		}
		ans = append(ans, mod)
	}
	if len(ans) == 1 {
		return ans[0], nil
	}
	return ans, nil
}

func (d *Driver) getLogicalSwitchPort(uuid string) (*LogicalSwitchPort, error) {
	cli, err := d.getNBCli()
	if err != nil {
		return nil, err
	}
	obj := &LogicalSwitchPort{
		UUID: uuid,
	}
	v, err := selectHelper(cli, "Logical_Switch_Port", &d.nbClientDBModel, obj)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil //nolint:nilnil
	}
	if ans, ok := v.(*LogicalSwitchPort); !ok {
		return nil, errors.New("failed to convert to LogicalSwitchPort")
	} else { //nolint
		return ans, nil
	}
}

func (d *Driver) getLogicalSwitchPortByName(name string) (*LogicalSwitchPort, error) {
	cli, err := d.getNBCli()
	if err != nil {
		return nil, err
	}
	obj := &LogicalSwitchPort{
		Name: name,
	}
	v, err := selectHelper(cli, "Logical_Switch_Port", &d.nbClientDBModel, obj)
	if err != nil {
		return nil, err
	}
	if ans, ok := v.(*LogicalSwitchPort); !ok {
		return nil, errors.New("failed to convert to LogicalSwitchPort")
	} else { //nolint
		return ans, nil
	}
}
