package main

import "github.com/emicklei/go-restful"

func setcors(c *restful.Container) {
	cors := restful.CrossOriginResourceSharing{
		AllowedDomains: []string{"*"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: true,
		Container:      c}
	c.Filter(cors.Filter)
	// Add container filter to respond to OPTIONS
	c.Filter(c.OPTIONSFilter)
}
