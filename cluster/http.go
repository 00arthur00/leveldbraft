package cluster

import (
	"net/http"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

type resource struct {
	raft Node
	log  hclog.Logger
}
type Msg struct {
	Code    int    `json:"code"`
	Data    string `json:"data"`
	Message string `json"message"`
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func newResouce(raft Node, log hclog.Logger) *resource {
	return &resource{raft, log}
}
func NewWebService(node Node, log hclog.Logger) *restful.WebService {
	r := newResouce(node, log)
	ws := &restful.WebService{}
	tags := []string{"raft leveldb"}

	ws.Path("/raft").Consumes(restful.MIME_JSON).Consumes(restful.MIME_JSON)

	ws.Route(ws.GET("/kv/{key}").To(r.get).
		Doc("get value of key").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.PathParameter("key", "key").DataType("string")).
		Writes(Msg{}).
		Returns(http.StatusOK, "ok", "").
		Returns(http.StatusBadRequest, "bad request", nil).
		Returns(http.StatusInternalServerError, "internal error", nil))

	ws.Route(ws.DELETE("/kv/{key}").To(r.delete).
		Doc("delete key").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(Msg{}).
		Returns(http.StatusOK, "ok", nil).
		Returns(http.StatusBadRequest, "bad request", nil).
		Returns(http.StatusInternalServerError, "internal error", nil))

	ws.Route(ws.PUT("/kv").To(r.set).
		Doc("set key/value").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Reads(KV{}, "key/value pair").
		Writes(Msg{}).
		Returns(http.StatusOK, "ok", nil).
		Returns(http.StatusBadRequest, "bad request", nil).
		Returns(http.StatusInternalServerError, "internal error", nil))

	ws.Route(ws.GET("/join").To(r.join).
		Doc("join the cluster").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Param(ws.QueryParameter("peer", "peer address")).
		Returns(http.StatusOK, "ok", nil).
		Returns(http.StatusBadRequest, "bad request", nil).
		Returns(http.StatusInternalServerError, "internal error", nil))

	ws.Route(ws.GET("/members").To(r.members).
		Doc("get cluster members").
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(raft.Configuration{}).
		Returns(http.StatusOK, "ok", nil).
		Returns(http.StatusBadRequest, "bad request", nil).
		Returns(http.StatusInternalServerError, "internal error", nil))

	return ws
}

func (r *resource) get(req *restful.Request, resp *restful.Response) {
	key := req.PathParameter("key")
	if key == "" {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, codeToMsg(http.StatusBadRequest))
		return
	}

	value, ok := r.raft.Get(key)
	if !ok {
		msg := codeToMsg(http.StatusBadRequest)
		resp.WriteHeaderAndEntity(http.StatusBadRequest, msg)
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, &Msg{
		Code:    http.StatusOK,
		Message: http.StatusText(http.StatusOK),
		Data:    value,
	})
}

func (r *resource) set(req *restful.Request, resp *restful.Response) {
	if !r.raft.IsLeader() {
		r.log.Error("http write to follower")
		resp.WriteHeaderAndEntity(http.StatusBadRequest, codeToMsg(http.StatusBadRequest))
		return
	}

	kv := KV{}
	if err := req.ReadEntity(&kv); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, codeToMsg(http.StatusBadRequest))
		return
	}

	if err := r.raft.Set(kv.Key, kv.Value); err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, codeToMsg(http.StatusInternalServerError))
		return
	}

	resp.WriteHeaderAndEntity(http.StatusOK, codeToMsg(http.StatusOK))
}

func (r *resource) delete(req *restful.Request, resp *restful.Response) {
	key := req.PathParameter("key")
	if key == "" {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, codeToMsg(http.StatusBadRequest))
		return
	}

	if err := r.raft.Delete(key); err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, codeToMsg(http.StatusInternalServerError))
		return
	}

	resp.WriteHeaderAndEntity(http.StatusOK, codeToMsg(http.StatusOK))
}

func (r *resource) join(req *restful.Request, resp *restful.Response) {
	if !r.raft.IsLeader() {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, &Msg{
			Code:    http.StatusBadRequest,
			Message: http.StatusText(http.StatusBadRequest),
			Data:    "cannot join to a non-leader portal",
		})
	}
	addr := req.QueryParameter("peer")
	if addr == "" {
		r.log.Error("error", "invalid peer addr")
		resp.WriteHeaderAndEntity(http.StatusBadRequest, codeToMsg(http.StatusBadRequest))
		return
	}
	if err := r.raft.Join(addr); err != nil {
		resp.WriteHeaderAndEntity(http.StatusInternalServerError, codeToMsg(http.StatusInternalServerError))
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, codeToMsg(http.StatusOK))
}

func (r *resource) members(req *restful.Request, resp *restful.Response) {

}
