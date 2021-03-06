package client

import (
	"bufio"
	"compress/gzip"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

type DockerClient struct {
	client *client.Client
}

func NewDockerClient(cli *cli.Context) (*DockerClient, error) {
	var c *client.Client
	var err error
	host := cli.String("host")
	if host == "" {
		c, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	} else {
		c, err = client.NewClientWithOpts(client.WithHost(host))
	}
	if err != nil {
		log.Panic("####connect docker error ", err)
	}
	return &DockerClient{
		client: c,
	}, err
}

//保存镜像
func (c *DockerClient) Save(cli *cli.Context) (err error) {
	t1 := time.Now()
	imagesVal := cli.StringSlice("images")
	pathVal := cli.String("path")
	images, fileName := resolveImages(imagesVal)
	for _, image := range images {
		log.Println("####开始拉取镜像:", image)
		c.pull(image)
	}
	log.Println("####开始保存镜像")
	ctx := context.Background()
	reader, err := c.client.ImageSave(ctx, images)
	if err != nil {
		log.Println("####read image error", err)
		return err
	}
	defer reader.Close()
	file, err := os.Create(path.Join(pathVal, fileName+".tar.gz"))
	if err != nil {
		log.Println("####create file error", err)
		return err
	}
	defer file.Close()
	writer := gzip.NewWriter(file)
	defer writer.Close()
	for {
		buff := make([]byte, 1024*10)
		i, err := reader.Read(buff)
		if err == io.EOF {
			break
		}
		writer.Write(buff[0:i])
	}
	t2 := time.Now()
	log.Printf("######耗时：%f s", t2.Sub(t1).Seconds())
	return err
}

//解析镜像
func resolveImages(imagesVal []string) ([]string, string) {
	format := time.Now().Format("20060102150405")
	if len(imagesVal) == 1 {
		filePath := path.Join("./", imagesVal[0])
		_, err := os.Stat(filePath)
		if err == nil {
			_, file := path.Split(imagesVal[0])
			split := strings.Split(file, ".")
			images := readFileImages(filePath)
			return images, split[0]
		}
	}
	return imagesVal, "images_" + format
}

func readFileImages(path string) []string {
	var images []string
	file, err := os.Open(path)
	if err != nil {
		return images
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		images = append(images, line)
	}
	return images
}

//拉取镜像
func (c *DockerClient) pull(image string) {
	ctx := context.Background()
	reader, err := c.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		log.Fatalf("####pull image %s failed %s", image, err)
		return
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)
}
