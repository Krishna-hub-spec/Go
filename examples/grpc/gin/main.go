package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	pb "github.com/gin-gonic/gin/examples/grpc/pb"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Set up a http setver.
	r := gin.Default()
	r.GET("/rest/n/:name", func(g *gin.Context) {
		name := g.Param("name")

		// Contact the server and print out its response.
		req := &pb.HelloRequest{Name: name}
		res, err := c.SayHello(g, req)
		if err != nil {
			g.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		} else {
			g.JSON(http.StatusOK, gin.H{
				"result": fmt.Sprint(res.Message),
			})
		}
	})

	// Run http server
	if err := r.Run(":8052"); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
