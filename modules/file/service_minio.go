package file

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Mininglamp-OSS/octo-lib/config"
	"github.com/Mininglamp-OSS/octo-lib/pkg/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// minioDefaultRegion is the region the MinIO SDK assumes when the server has
// not been told otherwise. Setting it explicitly avoids an unnecessary
// GetBucketLocation roundtrip on every request.
const minioDefaultRegion = "us-east-1"

// minioDefaultBucket is the bucket used when an object path does not carry a
// known prefix from `allowedMinioBuckets`.
const minioDefaultBucket = "file"

// minioBucketAlreadyOwnedByYou is the S3 error code returned when MakeBucket
// races with another caller that already created the same bucket under the
// same credentials. Treated as a benign no-op by `ensureBucket`.
const minioBucketAlreadyOwnedByYou = "BucketAlreadyOwnedByYou"

// readOnlyAnonymousPolicy is the bucket policy applied to every auto-created
// bucket: anonymous principals can GET objects, but uploads and deletes
// remain authenticated. Identical to the policy that the legacy `UploadFile`
// path used to inline.
const readOnlyAnonymousPolicy = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Allow",
		"Principal": {
			"AWS": ["*"]
		},
		"Action": ["s3:GetObject"],
		"Resource": ["arn:aws:s3:::%s/*"]
	}]
}`

// ServiceMinio 文件上传
type ServiceMinio struct {
	log.Log
	ctx            *config.Context
	downloadClient *http.Client

	// bucketLocks serializes ensureBucket calls per bucket so concurrent
	// uploads to a fresh bucket cannot double-create or race the
	// SetBucketPolicy step. The map is keyed by bucket name; values are
	// `*sync.Mutex` lazily inserted on first use via LoadOrStore. The map
	// itself is never deleted from — bucket count is bounded by the
	// allow-list, so growth is O(allowed buckets).
	bucketLocks sync.Map
}

// NewServiceMinio NewServiceMinio
func NewServiceMinio(ctx *config.Context) *ServiceMinio {
	return &ServiceMinio{
		Log: log.NewTLog("File"),
		ctx: ctx,
		downloadClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// ensureBucket guarantees that `bucket` exists on the MinIO server and has
// the read-only anonymous GET policy applied. Safe to call concurrently for
// the same bucket — a per-bucket mutex serializes the BucketExists / MakeBucket
// / SetBucketPolicy sequence so two parallel callers never double-create or
// race the policy update. The `BucketAlreadyOwnedByYou` S3 response is
// swallowed as a benign no-op for the case where another process (or another
// node sharing these credentials) won the create race.
func (sm *ServiceMinio) ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	mtxIface, _ := sm.bucketLocks.LoadOrStore(bucket, &sync.Mutex{})
	mtx := mtxIface.(*sync.Mutex)
	mtx.Lock()
	defer mtx.Unlock()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		sm.Error(fmt.Sprintf("检测 %s目录是否存在错误", bucket), zap.Error(err))
		return err
	}
	if exists {
		return nil
	}

	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: minioDefaultRegion}); err != nil {
		// Another caller (different process / different node sharing the
		// same credentials) may have created the bucket between our
		// BucketExists call and our MakeBucket call. Treat that specific
		// S3 response as a no-op rather than a hard failure.
		if minio.ToErrorResponse(err).Code == minioBucketAlreadyOwnedByYou {
			sm.Info("bucket already owned by us, skipping create", zap.String("bucket", bucket))
		} else {
			sm.Error(fmt.Sprintf("创建 %s目录失败", bucket), zap.Error(err))
			return err
		}
	}

	// Read-only public policy: allow anonymous download only. Upload and
	// delete go through authenticated server-side credentials.
	if err := client.SetBucketPolicy(ctx, bucket, fmt.Sprintf(readOnlyAnonymousPolicy, bucket)); err != nil {
		sm.Error("设置minio文件读写权限错误", zap.Error(err))
		return err
	}
	return nil
}

// UploadFile 上传文件
func (sm *ServiceMinio) UploadFile(filePath string, contentType string, contentDisposition string, copyFileWriter func(io.Writer) error) (map[string]interface{}, error) {
	buff := bytes.NewBuffer(make([]byte, 0))
	err := copyFileWriter(buff)
	if err != nil {
		sm.Error("复制文件内容失败！", zap.Error(err))
		return nil, err
	}

	ctx := context.Background()
	minioClient, err := sm.newClient()
	if err != nil {
		sm.Error("创建错误：", zap.Error(err))
		return nil, err
	}

	bucketName, fileName := splitBucketAndObject(filePath, minioDefaultBucket, allowedMinioBuckets)
	if err := sm.ensureBucket(ctx, minioClient, bucketName); err != nil {
		return nil, err
	}

	opts := minio.PutObjectOptions{ContentType: contentType, PartSize: 10 * 1024 * 1024}
	if contentDisposition != "" {
		opts.ContentDisposition = contentDisposition
	}
	n, err := minioClient.PutObject(ctx, bucketName, fileName, buff, int64(len(buff.Bytes())), opts)
	if err != nil {
		sm.Error("上传文件失败：", zap.Error(err))
		return map[string]interface{}{
			"path": "",
		}, err
	}
	return map[string]interface{}{
		"path": n.Key,
	}, err
}

func (sm *ServiceMinio) GetFile(ph string) (io.ReadCloser, string, error) {
	minioClient, err := sm.newClient()
	if err != nil {
		return nil, "", err
	}

	bucketName, objectPath := splitBucketAndObject(ph, minioDefaultBucket, allowedMinioBuckets)

	obj, err := minioClient.GetObject(context.Background(), bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, "", err
	}
	return obj, stat.ContentType, nil
}

func (sm *ServiceMinio) DownloadURL(ph string, filename string) (string, error) {
	minioConfig := sm.ctx.GetConfig().Minio
	result, _ := url.JoinPath(minioConfig.DownloadURL, ph)
	if strings.TrimSpace(filename) == "" {
		return result, nil
	}
	vals := url.Values{}
	encodedFilename := "UTF-8''" + url.QueryEscape(filename)
	vals.Set("response-content-disposition", fmt.Sprintf("attachment; filename*=%s", encodedFilename))
	return fmt.Sprintf("%s?%s", result, vals.Encode()), nil
}

// newClient builds a MinIO client pinned to the SDK's default region, which
// is what `mc` and the MinIO server itself ship with. Pinning the region
// here lets the SDK skip a GetBucketLocation pre-flight on every request.
func (sm *ServiceMinio) newClient() (*minio.Client, error) {
	minioConfig := sm.ctx.GetConfig().Minio
	uploadUl, _ := url.Parse(minioConfig.UploadURL)
	endpoint := uploadUl.Host
	useSSL := strings.HasPrefix(uploadUl.Scheme, "https")

	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioConfig.AccessKeyID, minioConfig.SecretAccessKey, ""),
		Secure: useSSL,
		Region: minioDefaultRegion,
	})
}

// rewriteToPublicHost rewrites the scheme/host of a presigned URL produced
// against the server-internal `UploadURL`/`DownloadURL` so that the link
// works from a browser. The internal URL is what the Go process talks to
// (often a Docker service name); the public URL is what end-user browsers
// resolve. When `publicBase` is empty the original URL is returned
// unchanged.
func rewriteToPublicHost(u *url.URL, publicBase string) *url.URL {
	publicBase = strings.TrimSpace(publicBase)
	if publicBase == "" {
		return u
	}
	parsed, err := url.Parse(strings.TrimRight(publicBase, "/"))
	if err != nil || parsed.Host == "" {
		return u
	}
	clone := *u
	clone.Host = parsed.Host
	clone.Scheme = parsed.Scheme
	return &clone
}

// PresignedPutURL generates a presigned PUT URL the browser can use to
// upload directly to MinIO, plus the matching anonymous GET URL for the
// resulting object. The target bucket is bootstrapped on first use via
// `ensureBucket` so a presigned PUT against a fresh deployment never lands
// on a NoSuchBucket response.
func (sm *ServiceMinio) PresignedPutURL(objectPath string, contentType string, contentDisposition string, expires time.Duration) (uploadURL string, downloadURL string, err error) {
	client, err := sm.newClient()
	if err != nil {
		return "", "", err
	}

	bucketName, objectKey := splitBucketAndObject(objectPath, minioDefaultBucket, allowedMinioBuckets)
	if objectKey == "" {
		return "", "", fmt.Errorf("空对象路径，无法生成预签名URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sm.ensureBucket(ctx, client, bucketName); err != nil {
		return "", "", fmt.Errorf("预签名上传前的目录引导失败: %w", err)
	}

	var presigned *url.URL
	if contentDisposition != "" || contentType != "" {
		headers := http.Header{}
		if contentType != "" {
			headers.Set("Content-Type", contentType)
		}
		if contentDisposition != "" {
			headers.Set("Content-Disposition", contentDisposition)
		}
		presigned, err = client.PresignHeader(ctx, http.MethodPut, bucketName, objectKey, expires, nil, headers)
	} else {
		presigned, err = client.PresignedPutObject(ctx, bucketName, objectKey, expires)
	}
	if err != nil {
		return "", "", fmt.Errorf("生成预签名URL失败: %w", err)
	}

	minioConfig := sm.ctx.GetConfig().Minio
	uploadURL = rewriteToPublicHost(presigned, minioConfig.UploadURL).String()

	dl, dlErr := sm.DownloadURL(objectPath, "")
	if dlErr != nil {
		sm.Warn("生成下载URL失败", zap.Error(dlErr))
	}
	return uploadURL, dl, nil
}

// PresignedGetURL generates a presigned GET URL with a Content-Disposition
// override so the browser saves the file under the correct user-facing
// filename. MinIO 默认 bucket 为公共读，但鉴权模式下也通过此方法签发。
func (sm *ServiceMinio) PresignedGetURL(objectPath string, filename string, disposition string, expires time.Duration) (string, error) {
	client, err := sm.newClient()
	if err != nil {
		return "", err
	}

	bucketName, objectKey := splitBucketAndObject(objectPath, minioDefaultBucket, allowedMinioBuckets)
	if objectKey == "" {
		return "", fmt.Errorf("空对象路径，无法生成预签名URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if disposition != "inline" {
		disposition = "attachment"
	}
	params := url.Values{}
	if filename != "" {
		encodedFilename := "UTF-8''" + rfc5987Encode(filename)
		params.Set("response-content-disposition", fmt.Sprintf("%s; filename*=%s", disposition, encodedFilename))
	} else {
		params.Set("response-content-disposition", disposition)
	}

	presigned, err := client.PresignedGetObject(ctx, bucketName, objectKey, expires, params)
	if err != nil {
		return "", fmt.Errorf("生成预签名GET URL失败: %w", err)
	}

	minioConfig := sm.ctx.GetConfig().Minio
	return rewriteToPublicHost(presigned, minioConfig.DownloadURL).String(), nil
}
