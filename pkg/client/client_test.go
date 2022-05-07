package client

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/NpoolPlatform/libent-cruder/pkg/cruder"
	constant "github.com/NpoolPlatform/service-template/pkg/const"

	// nolint:nolintlint
	testinit "github.com/NpoolPlatform/service-template/pkg/test-init"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/uuid"
)

var (
	once              sync.Once
	runByGithubAction bool
	err               error
)

func TestClient(t *testing.T) {
	once.Do(func() {
		runByGithubAction, err = strconv.ParseBool(os.Getenv("RUN_BY_GITHUB_ACTION"))
		if err := testinit.Init(); err != nil {
			fmt.Printf("cannot init test stub: %v\n", err)
		}
	})

	if err == nil && runByGithubAction {
		return
	}

	_, err := GetServiceTemplateInfoOnly(context.Background(),
		cruder.NewFilterConds().
			WithCond(constant.FieldID, cruder.EQ, structpb.NewStringValue(uuid.UUID{}.String())))
	if err != nil {
		t.Fatal(err)
	}
	// Here won't pass test due to we always test with localhost
}
