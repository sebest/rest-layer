package main

import (
	"log"
	"net/http"
	"net/url"

	"github.com/rs/cors"
	"github.com/rs/rest-layer"
	"github.com/rs/rest-layer/mem"
	"github.com/rs/rest-layer/schema"
)

var (
	// Define a user resource schema
	user = schema.Schema{
		"id": schema.Field{
			Required: true,
			// When a field is read-only, on default values or hooks can
			// set their value. The client can't change it.
			ReadOnly: true,
			// This is a field hook called when a new user is created.
			// The schema.NewID hook is a provided hook to generate a
			// unique id when no value is provided.
			OnInit: &schema.NewID,
			Validator: &schema.String{
				Regexp: "^[0-9a-f]{32}$",
			},
		},
		"created": schema.Field{
			Required:  true,
			ReadOnly:  true,
			OnInit:    &schema.Now,
			Validator: &schema.Time{},
		},
		"updated": schema.Field{
			Required: true,
			ReadOnly: true,
			OnInit:   &schema.Now,
			// The OnUpdate hook is called when the item is edited. Here we use
			// provided Now hook which just return the current time.
			OnUpdate:  &schema.Now,
			Validator: &schema.Time{},
		},
		// Define a name field as required with a string validator
		"name": schema.Field{
			Required: true,
			Validator: &schema.String{
				MaxLen: 150,
			},
		},
	}

	// Define a post resource schema
	post = schema.Schema{
		// schema.*Field are shortcuts for common fields (identical to users' same fields)
		"id":      schema.IDField,
		"created": schema.CreatedField,
		"updated": schema.UpdatedField,
		// Define a user field which references the user owning the post.
		// See bellow, the content of this field is enforced by the fact
		// that posts is a sub-resource of users.
		"user": schema.Field{
			Required: true,
			Validator: &schema.Reference{
				Path: "users",
			},
		},
		"public": schema.Field{
			Validator: &schema.Bool{},
		},
		// Sub-documents are handled via a sub-schema
		"meta": schema.Field{
			Schema: &schema.Schema{
				"title": schema.Field{
					Required: true,
					Validator: &schema.String{
						MaxLen: 150,
					},
				},
				"body": schema.Field{
					Validator: &schema.String{
						MaxLen: 100000,
					},
				},
			},
		},
	}
)

func main() {
	// Create a REST API root resource
	root := rest.New()

	// Add a resource on /users[/:user_id]
	users := root.Bind("users", rest.NewResource(user, mem.NewHandler(), rest.Conf{
		// We allow all REST methods
		// (rest.ReadWrite is a shortcut for []rest.Mode{Create, Read, Update, Delete, List})
		AllowedModes: rest.ReadWrite,
	}))

	// Bind a sub resource on /users/:user_id/posts[/:post_id]
	// and reference the user on each post using the "user" field of the posts resource.
	posts := users.Bind("posts", "user", rest.NewResource(post, mem.NewHandler(), rest.Conf{
		// Posts can only be read, created and deleted, not updated
		AllowedModes: []rest.Mode{rest.Read, rest.List, rest.Create, rest.Delete},
	}))

	// Add a friendly alias to public posts
	// (equivalent to /users/:user_id/posts?filter=public=true)
	posts.Alias("public", url.Values{"filter": []string{"public=true"}})

	// Create API HTTP handler for the resource graph
	api, err := rest.NewHandler(root)
	if err != nil {
		log.Fatalf("Invalid API configuration: %s", err)
	}

	// Add cors support
	h := cors.New(cors.Options{OptionsPassthrough: true}).Handler(api)

	// Bind the API under /api/ path
	http.Handle("/api/", http.StripPrefix("/api/", h))

	// Serve it
	log.Print("Serving API on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
