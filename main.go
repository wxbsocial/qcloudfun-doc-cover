package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/events"
)

var (
	SecretId    = Getenv("SecretId")
	SecretKey   = Getenv("SecretKey")
	QueueId     = Getenv("QueueId")
	ToBucket    = Getenv("ToBucket")
	ToRegion    = Getenv("ToRegion")
	DocPassword = Getenv("DocPassword")
	Zoom        = GetenvInt("Zoom", 100)
	ImageParams = Getenv("ImageParams")
)

func GetenvInt(key string, def int) int {
	value := Getenv(key)

	num, err := strconv.Atoi(value)
	if err != nil {
		return def
	}

	return num
}

func Getenv(key string) string {
	value := os.Getenv(key)
	fmt.Printf("%v=%v\n", key, value)
	return value
}

func handler(ctx context.Context, event events.COSEvent) (string, error) {

	for _, record := range event.Records {

		fmt.Printf("%+v\n", record)

		if strings.HasPrefix(record.Event.Name, "cos:ObjectCreated") {
			if err := createThumbs(ctx, record); err != nil {
				return "Fail", err
			}
		} else if strings.HasPrefix(record.Event.Name, "cos:ObjectRemove:Delete") {
			if err := deleteThumbs(ctx, record); err != nil {
				return "Fail", err
			}
		} else {
			fmt.Println("no handler")
		}

	}

	return "Success", nil
}

func deleteThumbs(ctx context.Context, record events.COSRecord) error {

	objectUrl, err := url.Parse(record.Object.Object.URL)
	if err != nil {
		return err
	}

	// u, _ := url.Parse("https://test-coz-private-1306409624.cos.ap-guangzhou.myqcloud.com")
	// 用于Get Service 查询，默认全地域 service.cos.myqcloud.com
	su, _ := url.Parse(fmt.Sprintf("https://cos.%v.myqcloud.com", ToRegion))

	bUrl, _ := url.Parse(fmt.Sprintf("https://%v.cos.%v.myqcloud.com", ToBucket, ToRegion))
	objectKey := strings.TrimPrefix(objectUrl.Path, "/")

	fmt.Printf("BucketURL:%v\n", bUrl)
	fmt.Printf("objectKey:%v\n", objectKey)

	b := &cos.BaseURL{BucketURL: bUrl, ServiceURL: su}
	// 1.永久密钥
	c := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  SecretId,
			SecretKey: SecretKey,
		},
	})

	filename := path.Base(objectKey)
	ext := path.Ext(filename)
	removedKey := strings.Replace(objectKey, filename, strings.TrimSuffix(filename, ext), 1)

	fmt.Printf("removedKey:%v\n", removedKey)
	var marker string
	opt := &cos.BucketGetOptions{
		Prefix:  removedKey,
		MaxKeys: 1000,
	}
	isTruncated := true
	for isTruncated {
		opt.Marker = marker
		v, _, err := c.Bucket.Get(ctx, opt)
		if err != nil {
			return err
		}
		for _, content := range v.Contents {
			_, err = c.Object.Delete(ctx, content.Key)
			if err != nil {
				return err
			}
		}
		isTruncated = v.IsTruncated
		marker = v.NextMarker
	}

	return nil
}

func createThumbs(ctx context.Context, record events.COSRecord) error {

	objectUrl, err := url.Parse(record.Object.Object.URL)
	if err != nil {
		return err
	}

	ciUrl := "https://" + strings.Replace(objectUrl.Host, ".cos.", ".ci.", 1)
	objectKey := strings.TrimPrefix(objectUrl.Path, "/")

	fmt.Printf("ciRawUrl:%v\n", ciUrl)
	fmt.Printf("objectKey:%v\n", objectKey)

	ci, _ := url.Parse(ciUrl)
	b := &cos.BaseURL{CIURL: ci}
	// 1.永久密钥
	c := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  SecretId,
			SecretKey: SecretKey,
		},
	})

	filename := path.Base(objectKey)
	ext := path.Ext(filename)
	relativePath := path.Join(strings.TrimSuffix(filename, ext), "thumb_${Page}.jpg")
	targetKey := strings.Replace(objectKey, filename, relativePath, 1)

	fmt.Printf("targetKey:%v\n", targetKey)

	createJobOpt := &cos.CreateDocProcessJobsOptions{
		Tag: "DocProcess",
		Input: &cos.DocProcessJobInput{
			Object: objectKey,
		},
		Operation: &cos.DocProcessJobOperation{
			Output: &cos.DocProcessJobOutput{
				Region: ToRegion,
				Object: targetKey,
				Bucket: ToBucket,
			},
			DocProcess: &cos.DocProcessJobDocProcess{
				TgtType:     "jpg",
				StartPage:   1,
				EndPage:     -1,
				DocPassword: DocPassword,
				Zoom:        Zoom,
				ImageParams: ImageParams,
			},
		},
		QueueId: QueueId,
	}
	res, _, err := c.CI.CreateDocProcessJobs(ctx, createJobOpt)
	if err != nil {

		return err
	}

	log.Printf("%+v\n", res)
	return nil
}

func main() {
	// Make the handler available for Remote Procedure Call by Cloud Function
	cloudfunction.Start(handler)
}
