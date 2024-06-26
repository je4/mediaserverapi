//go:build never

package grpc

import (
	"context"
	"net/http"
	"strings"

	"emperror.dev/errors"
	"github.com/je4/mediaserverapi/v2/pkg/grpcproto"
	mediaserverproto "github.com/je4/mediaserverproto/v2/pkg/mediaserver/proto"
	"github.com/je4/utils/v2/pkg/zLogger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mediaserverAPI struct {
	grpcproto.UnimplementedAPIServiceServer
	logger   zLogger.ZLogger
	dbClient mediaserverproto.DatabaseClient
}

func (api *mediaserverAPI) Ingest(ctx context.Context, item *grpcproto.IngestRequest) (*grpcproto.DefaultResponse, error) {

	var parent *mediaserverproto.ItemIdentifier
	if item.Parent != nil && *item.Parent != "" {
		parts := strings.SplitN(*item.Parent, "/", 2)
		if len(parts) != 2 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid parent %s", *item.Parent)
		}
		parent = &mediaserverproto.ItemIdentifier{
			Collection: parts[0],
			Signature:  parts[1],
		}
		resp, err := api.dbClient.ExistsItem(context.Background(), parent)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot check parent %s: %v", item.Parent, err)
		}
		if resp.GetStatus().Enum() != mediaserverproto.ResultStatus_OK.Enum() {
			return nil, status.Errorf(codes.InvalidArgument, "parent %s does not exist", item.Parent)
		}
	}
	if strings.Contains(item.Signature, "/") {
		return nil, status.Errorf(codes.InvalidArgument, "signature contains '/' character:  %s ", item.Signature)
	}

	result, err := api.dbClient.CreateItem(context.Background(), &mediaserverproto.NewItem{
		Identifier: &mediaserverproto.ItemIdentifier{
			Collection: item.GetCollection(),
			Signature:  item.GetSignature(),
		},
		Urn:           item.Urn,
		Public:        &item.Public,
		Parent:        parent,
		PublicActions: []byte(item.GetPublicActions()),
	})
	if err != nil {
		if status, ok := status.FromError(err); ok {
			if status.Code() == codes.AlreadyExists {
				NewResultMessage(c, http.StatusBadRequest, errors.Errorf("item %s/%s already exists", collection, item.Signature))
				return
			}
		}
		NewResultMessage(c, http.StatusInternalServerError, errors.Errorf("create item %s/%s: %v", collection, item.Signature, err))
		return
	}
	if result.Status.String() != mediaserverproto.ResultStatus_OK.String() {
		NewResultMessage(c, http.StatusInternalServerError, errors.Errorf("cannot create item: %s/%s: %s", collection, item.Signature, result.Message))
		return
	}
	c.JSON(http.StatusOK, HTTPResultMessage{
		Code:    http.StatusOK,
		Message: result.Message,
	})

}
