package pdg

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bots-for-me/common"
	"github.com/recoilme/pudge"
)

var log = common.Log

type Db struct {
	dbs        map[string]*pudge.Db
	path       string
	decoder    *gob.Decoder
	decoderMux sync.Mutex
	decoderBuf *bytes.Buffer
	encoder    *gob.Encoder
	encoderMux sync.Mutex
	encoderBuf *bytes.Buffer
	buf        *bytes.Buffer
	types      []interface{}
}

func New(path string, types ...interface{}) (this *Db, err error) {
	path = strings.TrimRightFunc(path, func(ch rune) bool { return ch == filepath.ListSeparator })
	// if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
	// 	err = common.Errorf(err)
	// 	return
	// }
	this = &Db{
		dbs:        make(map[string]*pudge.Db, 4),
		path:       path,
		types:      types,
		encoderBuf: &bytes.Buffer{},
		decoderBuf: &bytes.Buffer{},
	}
	this.encoder = gob.NewEncoder(this.encoderBuf)
	this.decoder = gob.NewDecoder(this.decoderBuf)
	for _, item := range types {
		if err = this.encoder.Encode(item); err != nil {
			err = common.Errorf("while encode %#v: %w", item, err)
			return
		}
		if _, err = this.decoderBuf.Write(this.encoderBuf.Bytes()); err != nil {
			err = common.Errorf("while write to decoder %#v: %w", item, err)
			return
		}
		this.encoderBuf.Reset()
		if err = this.decoder.Decode(item); err != nil {
			err = common.Errorf("while decode %#v: %w", item, err)
			return
		}
		dbCfg := *pudge.DefaultConfig
		dbCfg.SyncInterval = 1
		name := this.GetNameFor(item)
		if this.dbs[name], err = pudge.Open(filepath.Join(this.path, name), &dbCfg); err != nil {
			err = common.Errorf("while pudge.Open(): %w", err)
			return
		}
	}
	if err = this.Backup("./tmp"); err != nil {
		err = common.Errorf(err)
		return
	}
	return
}

func (this *Db) GetDbFor(item interface{}) *pudge.Db {
	return this.dbs[this.GetNameFor(item)]
}

func (this *Db) GetNameFor(item interface{}) string {
	return strings.TrimPrefix(fmt.Sprintf("%T", item), "*")
}

func (this *Db) Backup(tmpPath string) (err error) {
	log.Verbose("compacting...")
	// os.RemoveAll(tmpPath)
	if err = pudge.BackupAll(tmpPath); err != nil {
		err = common.Errorf(err)
		return
	}
	pudge.CloseAll()
	backPath := this.path + ".bak"
	// os.RemoveAll(backPath)
	if err = os.Rename(this.path, backPath); err != nil {
		err = common.Errorf(err)
		return
	}
	if err = os.Rename(filepath.Join("tmp", this.path), this.path); err != nil {
		os.Rename(backPath, this.path)
		err = common.Errorf(err)
		return
	}
	for _, item := range this.types {
		dbCfg := *pudge.DefaultConfig
		dbCfg.SyncInterval = 1
		name := this.GetNameFor(item)
		if this.dbs[name], err = pudge.Open(filepath.Join(this.path, name), &dbCfg); err != nil {
			err = common.Errorf("while pudge.Open(): %w", err)
			return
		}
	}
	os.RemoveAll(tmpPath)
	os.RemoveAll(backPath)
	log.Verbose("compacted")
	return
}

func (this *Db) Close() {
	log.Verbose("closing...")
	pudge.CloseAll()
	log.Verbose("closed")
}

func (this *Db) Encode(src interface{}) ([]byte, error) {
	this.encoderMux.Lock()
	defer this.encoderMux.Unlock()
	this.encoderBuf.Reset()
	if err := this.encoder.Encode(src); err != nil {
		return nil, common.Errorf("while Encode %#v: %w", src, err)
	}
	return append(make([]byte, 0, this.encoderBuf.Len()), this.encoderBuf.Bytes()...), nil
}

func (this *Db) Decode(src []byte, dst interface{}) (err error) {
	this.decoderMux.Lock()
	defer this.decoderMux.Unlock()
	this.decoderBuf.Reset()
	this.decoderBuf.Write(src)
	if err = this.decoder.Decode(dst); err != nil {
		err = common.Errorf("while Decode %#v: %w", dst, err)
	}
	return
}

func (this *Db) Get(id string, item interface{}) (found bool, err error) {
	tmp := []byte{}
	db := this.GetDbFor(item)
	if db == nil {
		err = common.Errorf("no Db for %#v", item)
		return
	}
	if err = db.Get(id, &tmp); err != nil {
		if err == pudge.ErrKeyNotFound {
			err = nil
			return
		} else {
			err = common.Errorf("while get %v %#v: %v", id, item, err)
			return
		}
	}
	found = true
	if err = this.Decode(tmp, item); err != nil {
		err = common.Errorf("%w", err)
		return
	}
	return
}

func (this *Db) Put(id string, item interface{}) (err error) {
	var encoded []byte
	if encoded, err = this.Encode(item); err != nil {
		err = common.Errorf("%w", err)
		return
	}
	db := this.GetDbFor(item)
	if db == nil {
		err = common.Errorf("no Db for %#v", item)
		return
	}
	if err = db.Set(id, encoded); err != nil {
		err = common.Errorf("while put %s %#v: %w", id, item, err)
	}
	return
}

func (this *Db) Del(id string, item interface{}) (err error) {
	name := this.GetNameFor(item)
	if err = this.dbs[name].Delete(id); err != nil {
		err = common.Errorf("while del %s %#v: %w", name, id, err)
	}
	return
}

// func (this *Db) GetDevice(id string) *hw.Device {
// 	device := hw.Device{}
// 	tmp := &[]byte{}
// 	if err := this.dbs[dbDevices].Get(id, tmp); err != nil {
// 		if err == pudge.ErrKeyNotFound {
// 			return nil
// 		} else {
// 			log.Fatal("while get device %#v: %v", id, err)
// 		}
// 	}
// 	this.Decode(*tmp, &device)
// 	return &device
// }

// func (this *Db) SetDevice(id string, device *hw.Device) {
// 	if err := this.dbs[dbDevices].Set(id, this.Encode(device)); err != nil {
// 		log.Fatal("while put device %#v: %v", id, err)
// 	}
// 	for _, battery := range device.Batteries {
// 		this.SetBattery(battery.Imei, battery)
// 	}
// }

// func (this *Db) SetBattery(id string, battery *hw.Battery) {
// 	if err := this.dbs[dbBatteries].Set(id, this.Encode(battery)); err != nil {
// 		log.Fatal("while put battery %#v: %v", id, err)
// 	}
// }

// func (this *Db) GetBattery(id string) *hw.Battery {
// 	battery := hw.Battery{}
// 	tmp := &[]byte{}
// 	if err := this.dbs[dbBatteries].Get(id, tmp); err != nil {
// 		if err == pudge.ErrKeyNotFound {
// 			return nil
// 		} else {
// 			log.Fatal("while get battery %#v: %v", id, err)
// 		}
// 	}
// 	this.Decode(*tmp, &battery)
// 	return &battery
// }

// func (this *Db) GetTask(id int) *hw.QueryEjectBatteryResult {
// 	task := hw.QueryEjectBatteryResult{}
// 	tmp := &[]byte{}
// 	if err := this.dbs[dbTasks].Get(id, tmp); err != nil {
// 		if err == pudge.ErrKeyNotFound {
// 			return nil
// 		} else {
// 			log.Fatal("while get task %#v: %v", id, err)
// 		}
// 	}
// 	this.Decode(*tmp, &task)
// 	return &task
// }

// func (this *Db) DeleteTask(id int) {
// 	if err := this.dbs[dbTasks].Delete(id); err != nil {
// 		if err != pudge.ErrKeyNotFound {
// 			log.Fatal("while get delete %#v: %v", id, err)
// 		}
// 	}
// }

// func (this *Db) SetTask(id int, task *hw.QueryEjectBatteryResult) {
// 	if err := this.dbs[dbTasks].Set(id, this.Encode(task)); err != nil {
// 		log.Fatal("while put task %#v: %v", id, err)
// 	}
// }
