package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"io"
	"net/http"
)

type BeaconClient struct {
	BeaconAddress string
}

func NewBeaconClient(url string) (*BeaconClient, error) {
	if url == "" {
		return nil, errors.New("empty Beacon Client URL provided")
	}
	return &BeaconClient{
		BeaconAddress: url,
	}, nil
}

func (cl *BeaconClient) FetchSidecar(ctx context.Context, slot uint64) (*derive.BlobsSidecar, error) {
	url := fmt.Sprintf("%s%s%d", cl.BeaconAddress, "/eth/v1/blobs/sidecar/", slot)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode > 204 {
		return nil, fmt.Errorf("status code %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var sidecar derive.BlobsSidecar
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return nil, err
	}

	return &sidecar, nil
}
