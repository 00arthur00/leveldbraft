package main

import (
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
)

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "raft leveldb",
			Description: "http portal for raft with backend leveldb store",
			Contact: &spec.ContactInfo{

				Name:  "yapo.yang",
				Email: "yang_yapo@126.com",
				URL:   "http://blog.yapo.fun/",
			},
			License: &spec.License{
				Name: "Apache License",
				URL:  "https://github.com/thanos-io/thanos/blob/master/LICENSE",
			},
			Version: "0.0.1",
		},
	}
}
func registerOpenAPI(c *restful.Container, prefix string) {
	cfg := restfulspec.Config{
		WebServices:                   c.RegisteredWebServices(), // you control what services are visible
		APIPath:                       prefix + "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}
	c.Add(restfulspec.NewOpenAPIService(cfg))
}
