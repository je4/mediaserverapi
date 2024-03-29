package api

import (
	"context"
	"crypto/tls"
	"emperror.dev/errors"
	"fmt"
	"github.com/bluele/gcache"
	"github.com/gin-gonic/gin"
	"github.com/je4/mediaserverapi/v2/pkg/api/docs"
	"github.com/je4/mediaserverdb/v2/pkg/mediaserverdbproto"
	"github.com/je4/utils/v2/pkg/zLogger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const BASEPATH = "/api/v1"

//	@title			Mediaserver API
//	@version		1.0
//	@description	Mediaserver API for managing collections and items
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	Jürgen Enge
//	@contact.url	https://ub.unibas.ch
//	@contact.email	juergen.enge@unibas.ch

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

func NewController(addr, extAddr string, tlsConfig *tls.Config, dbClient mediaserverdbproto.DBControllerClient, logger zLogger.ZLogger) (*controller, error) {
	u, err := url.Parse(extAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid external address '%s'", extAddr)
	}
	subpath := "/" + strings.Trim(u.Path, "/")

	// programmatically set swagger info
	docs.SwaggerInfo.Host = strings.TrimRight(fmt.Sprintf("%s:%s", u.Hostname(), u.Port()), " :")
	docs.SwaggerInfo.BasePath = "/" + strings.Trim(subpath+BASEPATH, "/")
	docs.SwaggerInfo.Schemes = []string{"https"}

	router := gin.Default()

	c := &controller{
		addr:     addr,
		router:   router,
		subpath:  subpath,
		cache:    gcache.New(100).LRU().Build(),
		logger:   logger,
		dbClient: dbClient,
	}
	if err := c.Init(tlsConfig); err != nil {
		return nil, errors.Wrap(err, "cannot initialize rest controller")
	}
	return c, nil
}

type controller struct {
	server   http.Server
	router   *gin.Engine
	addr     string
	subpath  string
	cache    gcache.Cache
	logger   zLogger.ZLogger
	dbClient mediaserverdbproto.DBControllerClient
}

func (ctrl *controller) Init(tlsConfig *tls.Config) error {
	v1 := ctrl.router.Group(BASEPATH)
	v1.GET("/ping", ctrl.ping)
	v1.GET("/collection/:collection", ctrl.collection)
	v1.PUT("/collection/:collection", ctrl.createItem)
	v1.GET("/storage/:storageid", ctrl.storage)
	v1.GET("/ingest", ctrl.getIngestItem)

	ctrl.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	//ctrl.router.StaticFS("/swagger/", http.FS(swaggerFiles.FS))

	ctrl.server = http.Server{
		Addr:      ctrl.addr,
		Handler:   ctrl.router,
		TLSConfig: tlsConfig,
	}

	if err := http2.ConfigureServer(&ctrl.server, nil); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (ctrl *controller) Start(wg *sync.WaitGroup) {
	go func() {
		wg.Add(1)
		defer wg.Done() // let main know we are done cleaning up

		if ctrl.server.TLSConfig == nil {
			fmt.Printf("starting server at http://%s\n", ctrl.addr)
			if err := ctrl.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				// unexpected error. port in use?
				fmt.Errorf("server on '%s' ended: %v", ctrl.addr, err)
			}
		} else {
			fmt.Printf("starting server at https://%s\n", ctrl.addr)
			if err := ctrl.server.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
				// unexpected error. port in use?
				fmt.Errorf("server on '%s' ended: %v", ctrl.addr, err)
			}
		}
		// always returns error. ErrServerClosed on graceful close
	}()
}

func (ctrl *controller) Stop() {
	ctrl.server.Shutdown(context.Background())
}

func (ctrl *controller) GracefulStop() {
	ctrl.server.Shutdown(context.Background())
}

// ping godoc
// @Summary      does pong
// @ID			 get-ping
// @Description  for testing if server is running
// @Tags         mediaserver
// @Produce      plain
// @Success      200  {string}  string
// @Router       /ping [get]
func (ctrl *controller) ping(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

type HTTPCollectionResultMessage struct {
	Name            string `json:"identifier,omitempty"`
	Description     string `json:"description,omitempty"`
	SignaturePrefix string `json:"signature_prefix,omitempty"`
	Secret          string `json:"secret,omitempty"`
	Public          string `json:"public,omitempty"`
	Jwtkey          string `json:"jwtkey,omitempty"`
	Storageid       string `json:"storageid,omitempty"`
}

// collection godoc
// @Summary      gets collection data
// @ID			 get-collection-by-name
// @Description  retrieves mediaserver collection information
// @Tags         mediaserver
// @Produce      json
// @Param		 collection path string true "collection name"
// @Success      200  {string}  HTTPCollectionResultMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /collection/{collection} [get]
func (ctrl *controller) collection(c *gin.Context) {
	collection := c.Param("collection")
	if collection == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no collection specified"))
		return
	}
	coll, err := ctrl.dbClient.GetCollection(context.Background(), &mediaserverdbproto.CollectionIdentifier{Collection: collection})
	if err != nil {
		if status, ok := status.FromError(err); ok {
			if status.Code() == codes.NotFound {
				NewResultMessage(c, http.StatusNotFound, errors.Wrapf(err, "collection %s not found", collection))
			} else {
				NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot get collection %s", collection))
			}
			return
		} else {
			NewResultMessage(c, http.StatusInternalServerError, errors.Wrap(err, "cannot get collection"))
			return
		}
	}
	c.JSON(http.StatusOK, HTTPCollectionResultMessage{
		Name:            coll.GetIdentifier().GetCollection(),
		Description:     coll.GetDescription(),
		SignaturePrefix: coll.GetSignaturePrefix(),
		Secret:          coll.GetSecret(),
		Public:          coll.GetPublic(),
		Jwtkey:          coll.GetJwtkey(),
		Storageid:       coll.GetStorageid(),
	})
}

type HTTPStorageResultMessage struct {
	Id         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Filebase   string `json:"filebase,omitempty"`
	Datadir    string `json:"datadir,omitempty"`
	Subitemdir string `json:"subitemdir,omitempty"`
	Tempdir    string `json:"tempdir,omitempty"`
}

// storage godoc
// @Summary      gets storage data
// @ID			 get-storage-by-id
// @Description  retrieves mediaserver storage information
// @Tags         mediaserver
// @Produce      json
// @Param		 storageid path string true "storage id"
// @Success      200  {string}  HTTPStorageResultMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /storage/{storageid} [get]
func (ctrl *controller) storage(c *gin.Context) {
	storageid := c.Param("storageid")
	if storageid == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no storage id specified"))
		return
	}
	storage, err := ctrl.dbClient.GetStorage(context.Background(), &mediaserverdbproto.StorageIdentifier{Id: storageid})
	if err != nil {
		if status, ok := status.FromError(err); ok {
			if status.Code() == codes.NotFound {
				NewResultMessage(c, http.StatusNotFound, errors.Wrapf(err, "storage %s not found", storageid))
			} else {
				NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot get storage %s", storageid))
			}
			return
		} else {
			NewResultMessage(c, http.StatusInternalServerError, errors.Wrap(err, "cannot get storage"))
			return
		}
	}
	c.JSON(http.StatusOK, HTTPStorageResultMessage{
		Id:         storage.GetId(),
		Name:       storage.GetName(),
		Filebase:   storage.GetFilebase(),
		Datadir:    storage.GetDatadir(),
		Subitemdir: storage.GetSubitemdir(),
		Tempdir:    storage.GetTempdir(),
	})
}

type CreateItemMessage struct {
	Signature     string `json:"signature"`
	Urn           string `json:"path"`
	Public        string `json:"public,omitempty"`
	Parent        string `json:"parent,omitempty"`
	PublicActions string `json:"public_actions,omitempty"`
}

// createItem godoc
// @Summary      creates new item
// @ID			 put-collection-item
// @Description  creates a new item for indexing
// @Tags         mediaserver
// @Produce      json
// @Param		 collection path string true "collection name"
// @Param 		 item       body CreateItemMessage true "new item to create"
// @Success      200  {object}  HTTPResultMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /collection/{collection} [put]
func (ctrl *controller) createItem(c *gin.Context) {
	collection := c.Param("collection")
	if collection == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no collection name specified"))
		return
	}
	var item CreateItemMessage
	if err := c.ShouldBindJSON(&item); err != nil {
		NewResultMessage(c, http.StatusBadRequest, errors.Wrap(err, "cannot bind item"))
		return
	}
	var parent *mediaserverdbproto.ItemIdentifier
	if item.Parent != "" {
		parts := strings.SplitN(item.Parent, "/", 2)
		if len(parts) != 2 {
			NewResultMessage(c, http.StatusBadRequest, errors.Errorf("invalid parent %s", item.Parent))
			return
		}
		parent = &mediaserverdbproto.ItemIdentifier{
			Collection: parts[0],
			Signature:  parts[1],
		}
		resp, err := ctrl.dbClient.ExistsItem(context.Background(), parent)
		if err != nil {
			NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot check parent %s", item.Parent))
			return
		}
		if resp.GetStatus().Enum() != mediaserverdbproto.ResultStatus_OK.Enum() {
			NewResultMessage(c, http.StatusBadRequest, errors.Errorf("parent %s does not exist", item.Parent))
			return
		}
	}
	if strings.Contains(item.Signature, "/") {
		NewResultMessage(c, http.StatusBadRequest, errors.Errorf("signature contains '/' character:  %s ", item.Signature))
		return
	}

	result, err := ctrl.dbClient.CreateItem(context.Background(), &mediaserverdbproto.NewItem{
		Identifier: &mediaserverdbproto.ItemIdentifier{
			Collection: collection,
			Signature:  item.Signature,
		},
		Urn:           item.Urn,
		Public:        []byte(item.Public),
		Parent:        parent,
		PublicActions: []byte(item.PublicActions),
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
	if result.Status.String() != mediaserverdbproto.ResultStatus_OK.String() {
		NewResultMessage(c, http.StatusInternalServerError, errors.Errorf("cannot create item: %s/%s: %s", collection, item.Signature, result.Message))
		return
	}
	c.JSON(http.StatusOK, HTTPResultMessage{
		Code:    http.StatusOK,
		Message: result.Message,
	})
}

type HTTPIngestItemMessage struct {
	Collection string `json:"collection"`
	Signature  string `json:"signature"`
	Urn        string `json:"path"`
}

// getIngestItem godoc
// @Summary      next ingest item
// @ID			 get-ingest-item
// @Description  gets next item for indexing
// @Tags         mediaserver
// @Produce      json
// @Success      200  {object}  HTTPIngestItemMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /ingest [get]
func (ctrl *controller) getIngestItem(c *gin.Context) {
	result, err := ctrl.dbClient.GetIngestItem(context.Background(), &emptypb.Empty{})
	if err != nil {
		if status, ok := status.FromError(err); ok {
			if status.Code() == codes.NotFound {
				NewResultMessage(c, http.StatusNotFound, errors.Wrapf(err, "no ingest item found"))
				return
			}
		}
		NewResultMessage(c, http.StatusInternalServerError, errors.Wrap(err, "cannot get ingest item"))
		return
	}
	c.JSON(http.StatusOK, HTTPIngestItemMessage{
		Collection: result.GetIdentifier().GetCollection(),
		Signature:  result.GetIdentifier().GetSignature(),
		Urn:        result.GetUrn(),
	})
}
