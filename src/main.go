package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/sirupsen/logrus"

	"app/clients"
	"app/log"
)

const HTTPStatusInvalidKey int = 566

func getUTCHash(timestamp time.Time) string {
	// FORMAT IS: "YYYY-MM"
	input := timestamp.Format("2006-01")
	salt := os.Getenv("SALT")
	key := fmt.Sprintf("%s_%s", input, salt)
	hasher := sha1.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func NewRequestID() string {
	hasher := sha1.New()
	hasher.Write([]byte(fmt.Sprint(time.Now().UTC().UnixNano())))
	return hex.EncodeToString(hasher.Sum(nil))
}

func verifyKey(key string) bool {
	switch key {
	case getUTCHash(time.Now().UTC()):
		fallthrough
	case getUTCHash(time.Now().UTC().AddDate(0, -1, 0)):
		fallthrough
	case getUTCHash((time.Now().UTC().AddDate(0, 1, 0))):
		return true
	default:
		return false
	}
}

var minioHost string
var minioPort int
var minioProtocol string
var minioDownloadBucket string
var minioStreamingBucket string
var minioSubtitleBucket string

func validateElasticSearchDocument(videoId string) bool {
	// validate video status.
	// always return true if there were some errors
	type docSrc struct {
		Deleted *bool `json:"deleted,omitempty"`
	}

	type doc struct {
		Index string  `json:"_index"`
		ID    *string `json:"_id,omitempty"`
		Src   *docSrc `json:"_source,omitempty"`
	}

	es, err := clients.GetElasticsearch()
	if err != nil {
		log.WithError(err).Error()
		return true
	}

	request := esapi.GetRequest{
		Index:        "heimdallr_streaming",
		DocumentType: "doc",
		DocumentID:   videoId,
		Source:       []string{"deleted"},
	}

	response, err := request.Do(context.Background(), es)
	if err != nil {
		log.WithError(err).Error()
		return true
	}

	if response == nil {
		log.WithFields(logrus.Fields{
			"DocumentID": videoId,
			"Index":      "heimdallr_streaming",
		}).Error("request returned nothing")
		return true
	}

	var results doc
	if response.StatusCode == 200 {
		json.NewDecoder(response.Body).Decode(&results)
		if results.Src != nil && results.Src.Deleted != nil && *results.Src.Deleted {
			return false
		}
	} else if response.StatusCode == 404 {
		log.WithFields(logrus.Fields{
			"DocumentID": videoId,
			"Index":      "heimdallr_streaming",
		}).Warn("request for non existing document")
		return false
	}
	return true
}

func cleanResponse(c *fiber.Ctx) {
	statusCode := c.Response().StatusCode()
	if statusCode < 200 || statusCode >= 400 {
		c.Response().SetBodyString("")
	}
	c.Response().Header.Set("Server", "B!gGo-Streamer")
	c.Response().Header.Del("X-Amz-Request-Id")
	c.Response().Header.Del("Vary")
	c.Response().Header.Set("Vary", "Origin")
}

func downloadHandler(c *fiber.Ctx) error {
	c.Request().Header.Set(log.HTTPRequestTrackingHeader, NewRequestID())
	log.HTTPRequest(c)

	// if !verifyKey(c.Params(("key"))) {
	// 	log.WithField("InvalidKey", c.Params("key")).Warning(c.Request())
	// 	return c.SendStatus(HTTPStatusInvalidKey)
	// }

	videoId := c.Params("video")
	if validate := validateElasticSearchDocument(videoId); !validate {
		return c.SendStatus(404)
	}
	url, _ := url.Parse(fmt.Sprintf("%s://%s:%d/%s/%s/%s",
		minioProtocol, minioHost, minioPort, minioDownloadBucket,
		c.Params("video"), c.Params("file")))

	c.Request().Header.Add("X-Real-IP", c.IP())
	c.Request().SetHost(url.Host)

	err := proxy.Do(c, url.String())
	if err == nil {
		statusCode := c.Response().StatusCode()
		if statusCode >= 200 && statusCode < 400 {
			c.Response().Header.Set("Content-Type", "video/mp4")
			c.Response().Header.Set("Content-Disposition", "attachment; filename=\""+c.Params("file")+"\"")
		}
		cleanResponse(c)
	} else {
		log.TraceHTTPRequest(c).WithError(err).Error(err)
	}
	return err
}

func m3u8Handler(c *fiber.Ctx) error {
	c.Request().Header.Set(log.HTTPRequestTrackingHeader, NewRequestID())
	log.HTTPRequest(c)

	if !verifyKey(c.Params(("key"))) {
		log.WithField("InvalidKey", c.Params("key")).Warning(c.Request())
		return c.SendStatus(HTTPStatusInvalidKey)
	}
	url, _ := url.Parse(fmt.Sprintf("%s://%s:%d/%s/%s/%s/stream.m3u8",
		minioProtocol, minioHost, minioPort, minioStreamingBucket,
		c.Params("video"), c.Params("quality")))

	c.Request().Header.Add("X-Real-IP", c.IP())
	c.Request().SetHost(url.Host)

	err := proxy.Do(c, url.String())
	if err == nil {
		//https://www.rfc-editor.org/rfc/rfc8216.html
		statusCode := c.Response().StatusCode()
		if statusCode >= 200 && statusCode < 400 {
			c.Response().Header.Set("Content-Type", "application/vnd.apple.mpegurl")
		}
		cleanResponse(c)
	} else {
		log.TraceHTTPRequest(c).WithError(err).Error(err)
	}
	return err
}

func segmentHandler(c *fiber.Ctx) error {
	c.Request().Header.Set(log.HTTPRequestTrackingHeader, NewRequestID())
	log.HTTPRequest(c)

	if !verifyKey(c.Params(("key"))) {
		log.WithField("InvalidKey", c.Params("key")).Warning(c.Request())
		return c.SendStatus(HTTPStatusInvalidKey)
	}
	url, _ := url.Parse(fmt.Sprintf("%s://%s:%d/%s/%s/%s/%s",
		minioProtocol, minioHost, minioPort, minioStreamingBucket,
		c.Params("video"), c.Params("quality"), c.Params("segment")))

	c.Request().Header.Add("X-Real-IP", c.IP())
	c.Request().SetHost(url.Host)

	err := proxy.Do(c, url.String())
	if err == nil {
		cleanResponse(c)
	} else {
		log.TraceHTTPRequest(c).WithError(err).Error(err)
	}
	return err
}

func subtitleHandler(c *fiber.Ctx) error {
	c.Request().Header.Set(log.HTTPRequestTrackingHeader, NewRequestID())
	log.HTTPRequest(c)
	c.Response().Body()

	if !verifyKey(c.Params(("key"))) {
		log.WithField("InvalidKey", c.Params("key")).Warning(c.Request())
		return c.SendStatus(HTTPStatusInvalidKey)
	}
	video_id := c.Params("video")
	url, _ := url.Parse(fmt.Sprintf("%s://%s:%d/%s/%s/%s_%s.vtt",
		minioProtocol, minioHost, minioPort, minioSubtitleBucket,
		video_id, video_id, c.Params("language")))

	c.Request().Header.Add("X-Real-IP", c.IP())
	c.Request().SetHost(url.Host)

	err := proxy.Do(c, url.String())
	if err == nil {
		statusCode := c.Response().StatusCode()
		if statusCode >= 200 && statusCode < 400 {
			c.Response().Header.Set("Content-Type", "application/vnd.apple.mpegurl")
		}
		cleanResponse(c)
	} else {
		log.TraceHTTPRequest(c).WithError(err).Error(err)
	}
	return err
}

func main() {
	minioHost = os.Getenv("MINIO_HOST")
	minioPort, _ = strconv.Atoi(os.Getenv("MINIO_PORT"))
	minioProtocol = os.Getenv("MINIO_PROTOCOL")
	minioDownloadBucket = os.Getenv("MINIO_DOWNLOAD_BUCKET")
	minioStreamingBucket = os.Getenv("MINIO_STREAMING_BUCKET")
	minioSubtitleBucket = os.Getenv("MINIO_SUBTITLE_BUCKET")

	app := fiber.New(fiber.Config{
		DisableKeepalive:             true,
		DisablePreParseMultipartForm: true,
		DisableStartupMessage:        true,
		ServerHeader:                 "B!gGo-Streamer",
		StreamRequestBody:            true,
	})

	app.Get("/download/:video/:file", downloadHandler)
	app.Get("/download/:key/:video/:file", downloadHandler)
	app.Get("/:key/:video/subtitle/:language", subtitleHandler)
	app.Get("/:key/:video/:quality", m3u8Handler)
	app.Get("/:key/:video/:quality/:segment", segmentHandler)

	app.Listen(":8000")
}
