package fabric

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

// OpenKV creates-or-opens a JetStream KeyValue bucket, backed by FileStorage
// so its contents survive a restart of the fabric on the same store dir.
func (f *Fabric) OpenKV(bucket string) (nats.KeyValue, error) {
	kv, err := f.js.KeyValue(bucket)
	if err == nil {
		return kv, nil
	}
	if !errors.Is(err, nats.ErrBucketNotFound) {
		return nil, fmt.Errorf("fabric: open kv %s: %v", bucket, err)
	}

	kv, err = f.js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:  bucket,
		Storage: nats.FileStorage,
	})
	if err != nil {
		return nil, fmt.Errorf("fabric: create kv %s: %v", bucket, err)
	}
	return kv, nil
}
