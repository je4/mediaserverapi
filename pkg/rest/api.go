package rest

import (
	"context"
	"crypto/tls"
	"emperror.dev/errors"
	"fmt"
	"github.com/bluele/gcache"
	"github.com/gin-gonic/gin"
	genericproto "github.com/je4/genericproto/v2/pkg/generic/proto"
	"github.com/je4/mediaserveraction/v2/pkg/actionCache"
	"github.com/je4/mediaserverapi/v2/pkg/rest/docs"
	mediaserverproto "github.com/je4/mediaserverproto/v2/pkg/mediaserver/proto"
	"github.com/je4/utils/v2/pkg/zLogger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
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

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func NewController(addr, extAddr string, tlsConfig *tls.Config, bearer string,
	dbClient mediaserverproto.DatabaseClient,
	actionControllerClient mediaserverproto.ActionClient,
	deleterControllerClient mediaserverproto.DeleterClient,
	logger zLogger.ZLogger) (*controller, error) {
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
		addr:                    addr,
		router:                  router,
		subpath:                 subpath,
		cache:                   gcache.New(100).LRU().Build(),
		logger:                  logger,
		dbClient:                dbClient,
		actionControllerClient:  actionControllerClient,
		deleterControllerClient: deleterControllerClient,
		bearer:                  bearer,
		actionParams:            map[string][]string{},
	}
	if err := c.Init(tlsConfig); err != nil {
		return nil, errors.Wrap(err, "cannot initialize rest controller")
	}
	return c, nil
}

type controller struct {
	server                  http.Server
	router                  *gin.Engine
	addr                    string
	subpath                 string
	cache                   gcache.Cache
	logger                  zLogger.ZLogger
	dbClient                mediaserverproto.DatabaseClient
	actionControllerClient  mediaserverproto.ActionClient
	deleterControllerClient mediaserverproto.DeleterClient
	bearer                  string
	actionParams            map[string][]string
}

func (ctrl *controller) Init(tlsConfig *tls.Config) error {
	v1 := ctrl.router.Group(BASEPATH)

	v1.GET("/ping", ctrl.ping)

	v1.Use(func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, HTTPResultMessage{
				Code:    http.StatusUnauthorized,
				Message: "no bearer token found",
			})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != ctrl.bearer {
			c.AbortWithStatusJSON(http.StatusUnauthorized, HTTPResultMessage{
				Code:    http.StatusUnauthorized,
				Message: "invalid bearer token",
			})
			return

		}
	})
	v1.GET("/collection", ctrl.collections)
	v1.GET("/collection/:collection", ctrl.collection)
	v1.PUT("/collection/:collection", ctrl.createItem)
	v1.GET("/cache/:collection/:signature/:action", ctrl.getCache)
	v1.GET("/cache/:collection/:signature/:action/*params", ctrl.getCache)
	v1.DELETE("/cache/:collection/:signature/:action", ctrl.deleteCache)
	v1.DELETE("/cache/:collection/:signature/:action/*params", ctrl.deleteCache)
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
	Name            string                    `json:"identifier,omitempty"`
	Description     string                    `json:"description,omitempty"`
	SignaturePrefix string                    `json:"signature_prefix,omitempty"`
	Secret          string                    `json:"secret,omitempty"`
	Public          string                    `json:"public,omitempty"`
	Jwtkey          string                    `json:"jwtkey,omitempty"`
	Storage         *HTTPStorageResultMessage `json:"storage,omitempty"`
}

// collection godoc
// @Summary      gets collection data
// @ID			 get-collection-by-name
// @Description  retrieves mediaserver collection information
// @Tags         mediaserver
// @Security 	 BearerAuth
// @Produce      json
// @Param		 collection path string true "collection name"
// @Success      200  {string}  HTTPCollectionResultMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      401  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /collection/{collection} [get]
func (ctrl *controller) collection(c *gin.Context) {
	collection := c.Param("collection")
	if collection == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no collection specified"))
		return
	}
	coll, err := ctrl.dbClient.GetCollection(context.Background(), &mediaserverproto.CollectionIdentifier{Collection: collection})
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
	stor := coll.GetStorage()
	storResult := &HTTPStorageResultMessage{
		Name:       stor.GetName(),
		Filebase:   stor.GetFilebase(),
		Datadir:    stor.GetDatadir(),
		Subitemdir: stor.GetSubitemdir(),
		Tempdir:    stor.GetTempdir(),
	}
	c.JSON(http.StatusOK, HTTPCollectionResultMessage{
		Name:            coll.GetName(),
		Description:     coll.GetDescription(),
		SignaturePrefix: coll.GetSignaturePrefix(),
		Secret:          coll.GetSecret(),
		Public:          coll.GetPublic(),
		Jwtkey:          coll.GetJwtkey(),
		Storage:         storResult,
	})
}

// collections godoc
// @Summary      gets collections data
// @ID			 get-collections
// @Description  retrieves mediaserver collections
// @Tags         mediaserver
// @Security 	 BearerAuth
// @Produce      json
// @Success      200  {array}   HTTPCollectionResultMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      401  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /collection [get]
func (ctrl *controller) collections(c *gin.Context) {

	colls, err := ctrl.dbClient.GetCollections(context.Background(), nil)
	if err != nil {
		NewResultMessage(c, http.StatusInternalServerError, errors.Wrap(err, "cannot get collection"))
		return
	}
	result := []HTTPCollectionResultMessage{}
	for {
		coll, err := colls.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			NewResultMessage(c, http.StatusInternalServerError, errors.Wrap(err, "cannot get collection"))
			return
		}

		stor := coll.GetStorage()
		storResult := &HTTPStorageResultMessage{
			Name:       stor.GetName(),
			Filebase:   stor.GetFilebase(),
			Datadir:    stor.GetDatadir(),
			Subitemdir: stor.GetSubitemdir(),
			Tempdir:    stor.GetTempdir(),
		}
		result = append(result, HTTPCollectionResultMessage{
			Name:            coll.GetName(),
			Description:     coll.GetDescription(),
			SignaturePrefix: coll.GetSignaturePrefix(),
			Secret:          coll.GetSecret(),
			Public:          coll.GetPublic(),
			Jwtkey:          coll.GetJwtkey(),
			Storage:         storResult,
		})
	}

	c.JSON(http.StatusOK, result)
}

type HTTPStorageResultMessage struct {
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
// @Security 	 BearerAuth
// @Produce      json
// @Param		 storageid path string true "storage id"
// @Success      200  {string}  HTTPStorageResultMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      401  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /storage/{storageid} [get]
func (ctrl *controller) storage(c *gin.Context) {
	storageid := c.Param("storageid")
	if storageid == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no storage id specified"))
		return
	}
	storage, err := ctrl.dbClient.GetStorage(context.Background(), &mediaserverproto.StorageIdentifier{Name: storageid})
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
		Name:       storage.GetName(),
		Filebase:   storage.GetFilebase(),
		Datadir:    storage.GetDatadir(),
		Subitemdir: storage.GetSubitemdir(),
		Tempdir:    storage.GetTempdir(),
	})
}

// CreateItemMessage represents the structure of the data required to create a new item.
// This structure is used when the client sends a request to create a new item in the collection.
type CreateItemMessage struct {
	// Signature is a unique identifier for the item within its collection.
	Signature string `json:"signature" example:"10_3931_e-rara-20425_20230519T104744_gen6_ver1.zip_10_3931_e-rara-20425_export_mets.xml"`

	// Urn represents the path of the item. It is used to locate the item within the system.
	Urn string `json:"path" example:"vfs://test/ub-reprofiler/mets-container-doi/bau_1/2023/9940561370105504/10_3931_e-rara-20425_20230519T104744_gen6_ver1.zip/10_3931_e-rara-20425/export_mets.xml"`

	// Public is an optional field that can be used to store any public data associated with the item.
	Public bool `json:"public,omitempty" example:"true"`

	// Parent is an optional field that represents the signature of the parent item, if any.
	// This is used to establish a parent-child relationship between items.
	Parent string `json:"parent,omitempty" example:"test/10_3931_e-rara-20425_20230519T104744_gen6_ver1.zip_10_3931_e-rara-20425_export_mets.xml"`

	// PublicActions is an optional field that can be used to store any public actions associated with the item.
	PublicActions string `json:"public_actions,omitempty"`

	// The type of ingest
	// * keep: ingest without copy of data
	// * copy: ingest with copy of data
	// * move: ingest with copy of data and deletion after copy
	// default: keep
	IngestType string `json:"ingest_type,omitempty" example:"copy" enums:"keep,copy,move"`
}

// createItem godoc
// @Summary      creates new item
// @ID			 put-collection-item
// @Description  creates a new item for indexing
// @Tags         mediaserver
// @Security 	 BearerAuth
// @Produce      json
// @Param		 collection path string true "collection name"
// @Param 		 item       body CreateItemMessage true "new item to create"
// @Success      200  {object}  HTTPResultMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      401  {object}  HTTPResultMessage
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
	var parent *mediaserverproto.ItemIdentifier
	if item.Parent != "" {
		parts := strings.SplitN(item.Parent, "/", 2)
		if len(parts) != 2 {
			NewResultMessage(c, http.StatusBadRequest, errors.Errorf("invalid parent %s", item.Parent))
			return
		}
		parent = &mediaserverproto.ItemIdentifier{
			Collection: parts[0],
			Signature:  parts[1],
		}
		resp, err := ctrl.dbClient.ExistsItem(context.Background(), parent)
		if err != nil {
			NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot check parent %s", item.Parent))
			return
		}
		if resp.GetStatus().Enum() != genericproto.ResultStatus_OK.Enum() {
			NewResultMessage(c, http.StatusBadRequest, errors.Errorf("parent %s does not exist", item.Parent))
			return
		}
	}
	if strings.Contains(item.Signature, "/") {
		NewResultMessage(c, http.StatusBadRequest, errors.Errorf("signature contains '/' character:  %s ", item.Signature))
		return
	}

	var ingestType mediaserverproto.IngestType
	switch item.IngestType {
	case "":
		ingestType = mediaserverproto.IngestType_KEEP
	case "keep":
		ingestType = mediaserverproto.IngestType_KEEP
	case "copy":
		ingestType = mediaserverproto.IngestType_COPY
	case "move":
		ingestType = mediaserverproto.IngestType_MOVE
	default:
		NewResultMessage(c, http.StatusBadRequest, errors.Errorf("invalid ingest type %s", item.IngestType))
		return
	}
	result, err := ctrl.dbClient.CreateItem(context.Background(), &mediaserverproto.NewItem{
		Identifier: &mediaserverproto.ItemIdentifier{
			Collection: collection,
			Signature:  item.Signature,
		},
		Urn:           item.Urn,
		Public:        &item.Public,
		Parent:        parent,
		PublicActions: []byte(item.PublicActions),
		IngestType:    &ingestType,
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
	if result.Status.String() != genericproto.ResultStatus_OK.String() {
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
// @Security 	 BearerAuth
// @Produce      json
// @Success      200  {object}  HTTPIngestItemMessage
// @Failure      400  {object}  HTTPResultMessage
// @Failure      401  {object}  HTTPResultMessage
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

func (ctrl *controller) getParams(mediaType string, action string) ([]string, error) {
	sig := fmt.Sprintf("%s::%s", mediaType, action)
	if params, ok := ctrl.actionParams[sig]; ok {
		return params, nil
	}
	resp, err := ctrl.actionControllerClient.GetParams(context.Background(), &mediaserverproto.ParamsParam{
		Type:   mediaType,
		Action: action,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get params for %s::%s", mediaType, action)
	}
	ctrl.logger.Debug().Msgf("params for %s::%s: %v", mediaType, action, resp.GetValues())
	ctrl.actionParams[sig] = resp.GetValues()
	return resp.GetValues(), nil
}

// createItem godoc
// @Summary      get cache metadata
// @ID			 get-collection-signature-action-params-cache
// @Description  gets cache data
// @Tags         mediaserver
// @Security 	 BearerAuth
// @Produce      json
// @Param		 collection path string true "collection name"
// @Param		 signature path string true "signature"
// @Param		 action path string true "action"
// @Param		 params path string true "params"
// @Success      200  {object}  any
// @Failure      400  {object}  HTTPResultMessage
// @Failure      401  {object}  HTTPResultMessage
// @Failure      404  {object}  HTTPResultMessage
// @Failure      500  {object}  HTTPResultMessage
// @Router       /cache/{collection}/{signature}/{action}/{params} [get]
func (ctrl *controller) getCache(c *gin.Context) {
	collection := c.Param("collection")
	if collection == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no collection name specified"))
		return
	}
	signature := c.Param("signature")
	if signature == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no signature specified"))
		return
	}
	action := c.Param("action")
	if action == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no action specified"))
		return
	}
	params := c.Param("params")

	item, err := ctrl.dbClient.GetItem(context.Background(), &mediaserverproto.ItemIdentifier{
		Collection: collection,
		Signature:  signature,
	})
	if err != nil {
		NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot get item %s/%s", collection, signature))
		return
	}

	ps := actionCache.ActionParams{}
	aparams, err := ctrl.getParams(item.GetMetadata().GetType(), action)
	if err != nil {
		NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot get params for %s::%s", collection, action))
		return
	}
	ps.SetString(params, aparams)

	resp, err := ctrl.dbClient.GetCache(context.Background(), &mediaserverproto.CacheRequest{
		Identifier: &mediaserverproto.ItemIdentifier{
			Collection: collection,
			Signature:  signature,
		},
		Action: action,
		Params: ps.String(),
	})
	if err != nil {
		NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot delete cache for %s/%s/%s/%s", collection, signature, action, params))
		return
	}
	c.JSON(http.StatusOK, resp.GetMetadata())
}

func (ctrl *controller) deleteCache(c *gin.Context) {
	collection := c.Param("collection")
	if collection == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no collection name specified"))
		return
	}
	signature := c.Param("signature")
	if signature == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no signature specified"))
		return
	}
	action := c.Param("action")
	if action == "" {
		NewResultMessage(c, http.StatusBadRequest, errors.New("no action specified"))
		return
	}
	params := c.Param("params")

	resp, err := ctrl.deleterControllerClient.DeleteCache(context.Background(), &mediaserverproto.CacheRequest{
		Identifier: &mediaserverproto.ItemIdentifier{
			Collection: collection,
			Signature:  signature,
		},
		Action: action,
		Params: params,
	})
	if err != nil {
		NewResultMessage(c, http.StatusInternalServerError, errors.Wrapf(err, "cannot delete cache for %s/%s/%s/%s", collection, signature, action, params))
		return
	}
	if resp.Status.String() != genericproto.ResultStatus_OK.String() {
		NewResultMessage(c, http.StatusInternalServerError, errors.Errorf("cannot delete cache for %s/%s/%s/%s: %s", collection, signature, action, params, resp.Message))
		return
	}
	c.JSON(http.StatusOK, HTTPResultMessage{
		Code:    int(resp.GetStatus()),
		Message: resp.Message,
	})
}
