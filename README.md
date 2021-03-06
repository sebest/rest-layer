# REST Layer [![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/rs/rest-layer) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/rs/rest-layer/master/LICENSE) [![build](https://img.shields.io/travis/rs/rest-layer.svg?style=flat)](https://travis-ci.org/rs/rest-layer)

REST Layer is a REST API framework heavily inspired by the excellent [Python Eve](http://python-eve.org). It lets you automatically generate a comprehensive, customizable, and secure REST API on top of any backend storage with no boiler plate code. You can focus on your business logic now.

Implemented as a `net/http` middleware, it plays well with other middlewares like [CORS](http://github.com/rs/cors).

REST Layer is an opinionated framework. Unlike many web frameworks, you don't directly control the routing. You just expose resources and sub-resources, the framework automatically figures what routes to generate behind the scene. You don't have to take care of the HTTP headers and response, JSON encoding, etc. either. rest handles HTTP conditional requests, caching, integrity checking for you. A powerful and extensible validation engine make sure that data comes pre-validated to you resource handlers. Generic resource handlers for MongoDB and other databases are also available so you have few to no code to write to make the whole system work.

<!-- TOC depth:6 withLinks:1 updateOnSave:1 orderedList:0 -->

- [REST Layer [![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/rs/rest-layer) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/rs/rest-layer/master/LICENSE) [![build](https://img.shields.io/travis/rs/rest-layer.svg?style=flat)](https://travis-ci.org/rs/rest-layer)](#rest-layer-godochttpimgshieldsiobadgegodoc-reference-bluesvgstyleflathttpsgodocorggithubcomrsrest-layer-licensehttpimgshieldsiobadgelicense-mit-redsvgstyleflathttpsrawgithubusercontentcomrsrest-layermasterlicense-buildhttpsimgshieldsiotravisrsrest-layersvgstyleflathttpstravis-ciorgrsrest-layer)
	- [Features](#features)
		- [Extensions](#extensions)
		- [Storage Handlers](#storage-handlers)
	- [Usage](#usage)
	- [Resource Configuration](#resource-configuration)
		- [Schema](#schema)
		- [Binding](#binding)
		- [Modes](#modes)
		- [Sub Resources](#sub-resources)
	- [Filtering](#filtering)
	- [Sorting](#sorting)
	- [Pagination](#pagination)
	- [Conditional Requests](#conditional-requests)
	- [Data Integrity and Concurrency Control](#data-integrity-and-concurrency-control)
	- [Data Validation](#data-validation)
		- [Nullable Values](#nullable-values)
		- [Extensible Data Validation](#extensible-data-validation)
	- [Timeout and Request Cancellation](#timeout-and-request-cancellation)
	- [Data Storage Handler](#data-storage-handler)
	- [Custom Response Sender](#custom-response-sender)
<!-- /TOC -->

## Features

- [x] Automatic handling of REST resource operations
- [ ] Full test coverage
- [x] Plays well with other `net/http` middlewares
- [x] Pluggable resources storage
- [x] Pluggable response sender
- [ ] GraphQL support
- [ ] Swagger Documentation
- [ ] Testing framework
- [x] Sub resources
- [ ] Cascading deletes on sub resources
- [x] Filtering
- [x] Sorting
- [x] Pagination
- [x] Aliasing
- [x] Custom business logic
- [ ] Event hooks
- [x] Field hooks
- [x] Extensible data validation and transformation
- [x] Conditional requests (Last-Modified / Etag)
- [x] Data integrity and concurrency control (If-Match)
- [x] Timeout and request cancellation thru [net/context](https://godoc.org/golang.org/x/net/context)
- [ ] Multi-GET
- [ ] Bulk inserts
- [x] Default and nullable values
- [ ] Per resource cache control
- [ ] Customizable authentication / authorization
- [ ] Projections
- [ ] Embedded resource serialization
- [x] Custom ID field
- [ ] Data versioning

### Extensions

- [x] [CORS](http://github.com/rs/cors)
- [ ] Method Override
- [ ] Gzip, Deflate
- [ ] JSONP
- [x] [X-Forwarded-For](https://github.com/sebest/xff)
- [x] [Rate Limiting](https://github.com/didip/tollbooth)
- [ ] Operations Log

### Storage Handlers

- [x] Memory (test only)
- [ ] MongoDB
- [ ] ElasticSearch
- [ ] Redis
- [ ] Google BigTable

## Usage

```go
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
```

Just run this code (or use the provided `cmd/demo/`):

	> go run cmd/demo/main.go
	2015/07/27 20:54:55 Serving API on http://localhost:8080

Using [HTTPie](http://httpie.org/), you can now play with your API.

First create a user:

```http
http POST :8080/api/users name="John Doe"

HTTP/1.1 201 Created
Content-Length: 155
Content-Location: /api/users/821d73ed48165b18462c820de9045ef6
Content-Type: application/json
Date: Mon, 27 Jul 2015 19:10:20 GMT
Etag: 1e18e148e1ff3ecdaae5ec03ac74e0e4
Last-Modified: Mon, 27 Jul 2015 19:10:20 GMT
Vary: Origin

{
    "id": "821d73ed48165b18462c820de9045ef6",
    "created": "2015-07-27T21:10:20.671003126+02:00",
    "updated": "2015-07-27T21:10:20.671003989+02:00",
    "name": "John Doe",
}
```

As you can see, the `id`, `created` and `updated` fields have been automatically generated by our `OnInit` field hooks.

Also notice the `Etag` and `Last-Modified` headers. Those guys allow data integrity and concurrency control through the use of the `If-Match` and `If-Unmodified-Since` headers. They can also serve for conditional requests using `If-None-Match` and `If-Modified-Since` headers.

Here is an example of conditional request:

```http
http :8080/api/users/821d73ed48165b18462c820de9045ef6 \
  If-Modified-Since:"Mon, 27 Jul 2015 19:10:20 GMT"

HTTP/1.1 304 Not Modified
Date: Mon, 27 Jul 2015 19:17:11 GMT
Vary: Origin
```

And here is a data integrity request following the [RFC-5789](http://tools.ietf.org/html/rfc5789) recommendations:

```http
http PATCH :8080/api/users/821d73ed48165b18462c820de9045ef6 \
  name="Someone Else" If-Match:invalid-etag

HTTP/1.1 412 Precondition Failed
Content-Length: 58
Content-Type: application/json
Date: Mon, 27 Jul 2015 19:33:27 GMT
Vary: Origin

{
    "code": 412,
    "fields": null,
    "message": "Precondition Failed"
}
```

Retry with the valid etag:

```http
http PATCH :8080/api/users/821d73ed48165b18462c820de9045ef6 \
  name="Someone Else" If-Match:1e18e148e1ff3ecdaae5ec03ac74e0e4

HTTP/1.1 200 OK
Content-Length: 159
Content-Type: application/json
Date: Mon, 27 Jul 2015 19:36:19 GMT
Etag: 7bb7a71b0f66197aa07c4c8fc9564616
Last-Modified: Mon, 27 Jul 2015 19:36:19 GMT
Vary: Origin

{
    "created": "2015-07-27T21:33:09.168492448+02:00",
    "id": "15a6918ac1acdf17433d2c3e074a610e",
    "name": "Someone Else",
    "updated": "2015-07-27T21:36:19.904545093+02:00"
}
```

Another cool thing is sub-resources. We've set our `posts` resource as a child of the `users` resource. This way we can handle ownership very easily as routes are constructed as `/users/:user_id/posts`.

Lets create a post:

```http
http POST :8080/api/users/821d73ed48165b18462c820de9045ef6/posts \
  meta:='{"title":"My first post"}'

HTTP/1.1 200 OK
Content-Length: 212
Content-Type: application/json
Date: Mon, 27 Jul 2015 19:46:55 GMT
Etag: 307ae92df6c3dd54847bfc7d72422e07
Last-Modified: Mon, 27 Jul 2015 19:46:55 GMT
Vary: Origin

{
    "created": "2015-07-27T21:46:55.355857401+02:00",
    "id": "251511a70447b5914e835b8a4d357397",
    "meta": {
        "title": "My first post"
    },
    "updated": "2015-07-27T21:46:55.355857989+02:00",
    "user": "821d73ed48165b18462c820de9045ef6"
}
```

Notice how the `user` field has been set with the user id provided in the route, that's pretty cool, huh?

We defined that we can create posts but we can't modify them, lets verify that:

```http
http PUT :8080/api/users/821d…/posts/251511a70447b5914e835b8a4d357397 \
  private=true

HTTP/1.1 405 Method Not Allowed
Content-Length: 53
Content-Type: application/json
Date: Mon, 27 Jul 2015 19:50:33 GMT
Vary: Origin

{
    "code": 405,
    "fields": null,
    "message": "Invalid method"
}
```

Let's list posts for that user now:

```http
http :8080/api/users/821d73ed48165b18462c820de9045ef6/posts
HTTP/1.1 200 OK
Content-Length: 257
Content-Type: application/json
Date: Mon, 27 Jul 2015 19:51:46 GMT
Vary: Origin
X-Page: 1
X-Total: 1

[
    {
        "_etag": "307ae92df6c3dd54847bfc7d72422e07",
        "created": "2015-07-27T21:46:55.355857401+02:00",
        "id": "251511a70447b5914e835b8a4d357397",
        "meta": {
            "title": "My first post"
        },
        "updated": "2015-07-27T21:46:55.355857989+02:00",
        "user": "821d73ed48165b18462c820de9045ef6"
    }
]
```

Notice the added `_etag` field. This is to let you get etags of multiple items without having to `GET` each one of them.

## Resource Configuration

For REST Layer to be able to expose resources, you have to first define what fields the resource contains and where to bind it in the REST API URL namespace.

### Schema

Resource field configuration is performed thru the [schema](https://godoc.org/github.com/rs/rest-layer/schema) package. A schema is a map of field name pointing to field definition. The field definition contains the following properties:

| Property    | Description
| ----------- | -------------
| `Required`  | If `true`, the field must be provided when the resource is created and can't be set to `null`. The client may be` able` to omit a required field if a `Default` or a hook sets its content.
| `ReadOnly`  | If `true`, the field can not be set by the client, only a `Default` or a hook can alter its value. You may specify a value for a read-only field in your mutation request if the value is equal to the old value, REST Layer won't complain about it. This let your client to `PUT` the same document it `GET` without having to take care of removing read-only fields.
| `Default`   | The value to be set when resource is created and the client didn't provided a value for the field. The content of` this` variable must still pass validation.
| `OnInit`    | A function to be executed when the resource is created. The function gets the current value of the field (a`fter` `Default` has been set if any) and returns the new value to be set.
| `OnUpdate`  | A function to be executed when the resource is updated. The function gets the current (updated) value of the fi`eld` and returns the new value to be set.
| `Validator` | A `schema.FieldValidator` to validate the content of the field.
| `Schema`    | An optional sub schema to validate hierarchical documents.

REST Layer comes with a set of validators. You can add your own by implementing the `schema.FieldValidator` interface. Here is the list of provided validators:

| Validator          | Description
| ------------------ | -------------
| `schema.String`    | Ensures the field is a string
| `schema.Integer`   | Ensures the field is an integer
| `schema.Float`     | Ensures the field is a float
| `schema.Bool`      | Ensures the field is a Boolean
| `schema.Array`     | Ensures the field is an array
| `schema.Dict`      | Ensures the field is a dict
| `schema.Time`      | Ensures the field is a datetime
| `schema.Reference` | Ensures the field contains a reference to another _existing_ API item
| `schema.AnyOf`     | Ensures that at least one sub-validator is valid
| `schema.AllOf`     | Ensures that at least all sub-validators are valid

Some common hook handler to be used with `OnInit` and `OnUpdate` are also provided:

| Hook           | Description
| -------------- | -------------
| `schema.Now`   | Returns the current time ignoring the input (current) value.
| `schema.NewID` | Returns a unique identified if input value is `nil`.

Some common field configuration are also provided as variable:

| Field Config          | Description
| --------------------- | -------------
| `schema.IDField`      | A required, read-only field with `schema.NewID` set as `OnInit` hook and a `schema.String` va`lidator.
| `schema.CreatedField` | A required, read-only field with `schema.Now` set on `OnInit` hook with a `schema.Time` validator
| `schema.UpdatedField` | A required, read-only field with `schema.Now` set on `OnInit` and `OnUpdate` hooks with a `schema.Time` validator.

Here is an example of schema declaration:

```go
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

```

### Binding

Now you just need to bind this schema at a specific endpoint on the `rest.Handler` object:

```go
root := rest.New()
posts := root.Bind("posts", rest.NewResource(post, mem.NewHandler(), rest.DefaultConf)
```

This tells the `rest.Handler` to bind the `post` schema at the `posts` endpoint. The resource collection URL is then `/posts` and item URLs are `/posts/<post_id>`.

The `rest.DefaultConf` variable is a pre-defined `rest.Conf` type with sensible default. You can customize the resource behavoir using a custom configuration.

The `rest.Conf` type has the following customizable properties:

| Property                 | Description
| ------------------------ | -------------
| `AllowedModes`           | A list of `rest.Mode` allowed for the resource.
| `PaginationDefaultLimit` | If set, pagination is enabled by default with a number of item per page defined here.


### Modes

REST Layer handles mapping of HTTP methods to your resource URLs automatically. With REST, there is two kind of resource URL pathes: collection and item URLs. Collection URLs (`/<resource>`) are pointing to the collection of items while item URL (`/<resource>/<item_id>`) points to a specific item in that collection. HTTP methods are used to perform CRUDL operations on those resource.

You can easily dis/allow operation on a per resource basis using `rest.Conf` `AllowedModes` property. The use of modes instead of HTTP methods in the configuration adds a layer of abstraction necessary to handle specific cases like `PUT` HTTP method performing a `create` if the specified item does not exist or a `replace` if it does. This gives you precise control of what you want to allow or not.

Modes are passed as configuration to resources as follow:

```go
users := api.Bind("users", rest.NewResource(user, mem.NewHandler(), rest.Conf{
	AllowedModes: []rest.Mode{rest.Read, rest.List, rest.Create, rest.Delete},
}))
```

The following table shows how REST layer map CRUDL operations to HTTP methods and `modes`:

| Mode      | HTTP Method | Context    | Description
| --------- | ----------- | ---------- | -------------
| `Read`    | GET         | Item       | Get an individual item by its ID
| `List`    | GET         | Collection | List/find items using filters and sorts
| `Create`  | POST        | Collection | Create an item letting the system generate its ID
| `Create`  | PUT         | Item       | Create an item by choosing its ID
| `Update`  | PATCH       | Item       | Partialy modify the item following [RFC-5789](http://tools.ietf.org/html/rfc5789)
| `Replace` | PUT         | Item       | Replace the item by a new on
| `Delete`  | DELETE      | Item       | Delete the item by its ID
| `Clear`   | DELETE      | Collection | Delete all items from the collection matching the context and/or filters

### Sub Resources

Sub resources can be used to express a one-to-may parent-child relationship between two resources. A sub-resource is automatically filtered by it's parent.

To create a sub-resource, you bind you resource on the object returned by the binding of the parent resource. For instance, here we bind a `comments` resource to a `posts` resource:

```go
posts := root.Bind("posts", rest.NewResource(post, mem.NewHandler(), rest.DefaultConf)
// Bind comment as sub-resource of the posts resource
posts.Bind("comments", "post", rest.NewResource(comment, mem.NewHandler(), rest.DefaultConf)
```

The second argument `"post"` defines the field in the `comments` resource that refers to the parent. This field must be present in the resource and the backend storage must support filtering on it. As a result, we get a new hierarchical route as follow:

	/posts/:post_id/comments[/:comment_id]

When performing a `GET` on `/posts/:post_id/comments`, it is like adding the filter `{"post":"<post_id>"}` to the request to comments resource.

## Filtering

To filter resources, use the `filter` query-string parameter. The format of the parameter is inspired the [MongoDB query format](http://docs.mongodb.org/manual/tutorial/query-documents/). The `filter` parameter can be used with `GET` and `DELETE` methods on collection URLs.

To specify equality condition, use the query `{<field>: <value>}` to select all items with `<field>` equal `<value>`. REST Layer will complain with a `422` HTTP error if any field queried is not defined in the resource schema or is using an operator incompatible with field type (i.e.: `$lt` on a string field).

A query can specify conditions for more than one field. Implicitly, a logical `AND` conjunction connects the clauses so that the query selects the items that match all the conditions.

Using the the `$or` operator, you can specify a compound query that joins each clause with a logical `OR` conjunction so that the query selects the items that match at least one condition.

In the following example, the query document selects all documents in the collection where the field `quantity` has a value greater than (`$gt`) `100` or the value of the `price` field is less than (`$lt`) `9.95`:

```json
{"$or": [{"quantity": {"$gt": 100}}, {"price": {"$lt": 9.95}}]}
```

Match on sub-fields is performed thru field path separated by dots. This example shows an exact match on the subfields `country` and `city` of the `address` sub-document:

```json
{"address.country": "France", "address.city": "Paris"}
```

Some operators can change the type of match. For instance `$in` can be used to match a field against several values. For instance, to select all items with the `type` field equal either `food` or `snacks`, use the following query:

```json
{"type": {"$in": ["food", "snacks"]}}
```

The opposite `$nin` is also available.

The following numeric comparisons operators are supported: `$lt`, `$lte`, `$gt`, `$gte`.

## Sorting

Sorting is of resource items is defined thru the `sort` query-string parameter. The `sort` value is a list of resource's fields separated by comas (,). To invert a field's sort, you can prefix it's with a minus (-) character.

Here we sort the result by ascending quantity and descending date:

	sort=quantity,-created

## Pagination

Pagination is supported on collection URLs using `page` and `limit` query-string parameters. If you don't define a default pagination limit using `PaginationDefaultLimit` resource configuration parameter, the resource won't be paginated until you provide the `limit` query-string parameter.

## Conditional Requests

Each stored resource provides information on the last time it was updated (`Last-Modified`), along with a hash value computed on the representation itself (`ETag`). These headers allow clients to perform conditional requests by using the `If-Modified-Since` header:

```http
> http :8080/users/521d6840c437dc0002d1203c If-Modified-Since:'Wed, 05 Dec 2012 09:53:07 GMT'
HTTP/1.1 304 Not Modified
```

or the If-None-Match header:

```http
$ http :8080/users/521d6840c437dc0002d1203c If-None-Match:1234567890123456789012345678901234567890
HTTP/1.1 304 Not Modified
```

## Data Integrity and Concurrency Control

API responses include a `ETag` header which also allows for proper concurrency control. An `ETag` is a hash value representing the current state of the resource on the server. Clients may choose to ensure they update (`PATCH` or `PUT`) or delete (`DELETE`) a resource in the state they know it by providing the last known `ETag` for that resource. This prevents overwriting items with obsolete versions.

Consider the following workflow:

```http
$ http PATCH :8080/users/521d6840c437dc0002d1203c If-Match:1234567890123456789012345678901234567890 name='John Doe'
HTTP/1.1 412 Precondition Failed
```

What went wrong? We provided a `If-Match` header with the last known `ETag`, but it’s value did not match the current `ETag` of the item currently stored on the server, so we got a 412 Precondition Failed.

When this happen, it's up to the client to decide to inform the user of the error and/or refetch the latest version of the document to get the lattest `ETag` before retrying the operation.

```http
$ http PATCH :8080/users/521d6840c437dc0002d1203c If-Match:80b81f314712932a4d4ea75ab0b76a4eea613012 name='John Doe'
Etag: 7bb7a71b0f66197aa07c4c8fc9564616
Last-Modified: Mon, 27 Jul 2015 19:36:19 GMT
```

This time the update operation has been accepted and we've got a new `ETag` for the updated resource.

Concurrency control header `If-Match` can be used with all mutation methods on item URLs: `PATCH` (update), `PUT` (replace) and `DELETE` (delete).

## Data Validation

Data validation is provided out-of-the-box. Your configuration includes a schema definition for every resource managed by the API. Data sent to the API to be inserted/updated will be validated against the schema, and a resource will only be updated if validation passes.

```http
> http  :8080/api/users name:=1 foo=bar
HTTP/1.1 422 status code 422
Content-Length: 110
Content-Type: application/json
Date: Thu, 30 Jul 2015 21:56:39 GMT
Vary: Origin

{
    "code": 422,
    "message": "Document contains error(s)"
    "issues": {
        "foo": [
            "invalid field"
        ],
        "name": [
            "not a string"
        ]
    },
}
```

In the example above, the document did not validate so the request has been rejected with description of the errors for each fields.

### Nullable Values

To allow `null` value in addition the field type, you can use `schema.AnyOf` validator:

```go
"nullable_field": schema.AnyOf{
	schema.String{},
	schema.Null{},
}
```

### Extensible Data Validation

It is very easy to add new validators. You just need to implement the `schema.FieldValidator`:

```go
type FieldValidator interface {
	Validate(value interface{}) (interface{}, error)
}
```

The `Validate` method takes the value as argument and must either return the value back with some eventual transformation or an `error` if the validation failed.

Your validator may also implement the optional `schema.Compiler` interface:

```go
type Compiler interface {
	Compile() error
}
```

When a field validator implements this interface, the `Compile` method is called at the binding. It's a good place to pre-compute some data (i.e.: compile regexp) and verify validator configuration. If validator configuration contains issue, the `Compile` method must return an error, so the binding will generate un fatal error.

## Timeout and Request Cancellation

REST Layer handles client request cancellation using [net/context](https://golang.org/x/net/context). In case the client closes the connection before the server has finish processing the request, the context is canceled. This context is passed to the resource handler so it can listen for those cancelations and stop the processing (see [Data Storage Handler](#data-storage-handler) section for more information about how to implement resource handlers.

Timeout is implement the same way. If a timeout is set at server level through `rest.Handler` `RequestTimeout` property or if the `timeout` query-string parameter is passed with a duration value compatible with `time.ParseDuration`, the context will be set with a deadline set to that value.

When a request is stopped because the client closed the connection, the response HTTP status is set to `499 Client Closed Request` (for logging purpose). When a timeout is set and the request has reached this timeout, the response HTTP status is set to `509 Gateway Timeout`.

## Data Storage Handler

REST Layer doesn't handle storage of resources directly. A `mem.MemoryHandler` is provided as an example but should be used for testing only.

A resource handler is easy to write though. Some handlers for popular databases are available (soon), but you may want to write your own to put an API in front of anything you want. It is very easy to write a data storage handler, you just need to implement the `rest.ResourceHandler` interface:

```go
type ResourceHandler interface {
	Find(lookup *rest.Lookup, page, perPage int, ctx context.Context) (*rest.ItemList, *rest.Error)
	Insert(items []*Item, ctx context.Context) *Error
	Update(item *Item, original *Item, ctx context.Context) *Error
	Delete(item *rest.Item, ctx context.Context) *rest.Error
	Clear(lookup *rest.Lookup, ctx context.Context) (int, *rest.Error)
}
```

Mutation methods like `Update` and `Delete` must ensure they are atomically mutating the same item as specified in argument by checking their `ETag` (the stored `ETag` must match the `ETag` of the provided item). In case the handler can't guarantee that, the storage must be left untouched, and a `rest.ConflictError` must be returned.

If the the operation not immediate, the method must listen for cancellation on the passed `ctx`. If the operation is stopped due to context cancellation, the function must return the result of the `rest.ContextError()` with the `ctx.Err()` as argument. See [this blog post](https://blog.golang.org/context) for more information about how `net/context` works.

See [rest.ResourceHandler](https://godoc.org/github.com/rs/rest-layer#ResourceHandler) documentation for more information on resource handler implementation details.

## Custom Response Sender
