package files

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	http2 "github.com/aihop/gopanel/utils/http"
)

type WriteCounter struct {
	Total   uint64
	Written uint64
	Key     string
	Name    string
}

type Process struct {
	Total   uint64  `json:"total"`
	Written uint64  `json:"written"`
	Percent float64 `json:"percent"`
	Name    string  `json:"name"`
}

func (w *WriteCounter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.Written += uint64(n)
	w.SaveProcess()
	return n, nil
}

func (w *WriteCounter) SaveProcess() {
	percentValue := 0.0
	if w.Total > 0 {
		percent := float64(w.Written) / float64(w.Total) * 100
		percentValue, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", percent), 64)
	}
	process := Process{
		Total:   w.Total,
		Written: w.Written,
		Percent: percentValue,
		Name:    w.Name,
	}
	by, _ := json.Marshal(process)
	if percentValue < 100 {
		if err := global.CACHE.Set(w.Key, string(by)); err != nil {
			global.LOG.Errorf("save cache error, err %s", err.Error())
		}
	} else {
		if err := global.CACHE.SetWithTTL(w.Key, string(by), time.Second*time.Duration(10)); err != nil {
			global.LOG.Errorf("save cache error, err %s", err.Error())
		}
	}
}

func (f FileOp) DownloadFileWithProcess(url, dst, key string, ignoreCertificate bool) error {
	client := &http.Client{}
	if ignoreCertificate {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	request.Header.Set("Accept-Encoding", "identity")
	resp, err := client.Do(request)
	if err != nil {
		global.LOG.Errorf("get download file [%s] error, err %s", dst, err.Error())
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		global.LOG.Errorf("create download file [%s] error, err %s", dst, err.Error())
		return err
	}
	go func() {
		counter := &WriteCounter{}
		counter.Key = key
		if resp.ContentLength > 0 {
			counter.Total = uint64(resp.ContentLength)
		}
		counter.Name = filepath.Base(dst)
		if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
			global.LOG.Errorf("save download file [%s] error, err %s", dst, err.Error())
		}
		out.Close()
		resp.Body.Close()

		value, err := global.CACHE.Get(counter.Key)
		if err != nil {
			global.LOG.Errorf("get cache error,err %s", err.Error())
			return
		}
		process := &Process{}
		_ = json.Unmarshal(value, process)
		process.Percent = 100
		process.Name = counter.Name
		process.Total = process.Written
		by, _ := json.Marshal(process)
		if err := global.CACHE.SetWithTTL(counter.Key, string(by), time.Second*time.Duration(10)); err != nil {
			global.LOG.Errorf("save cache error, err %s", err.Error())
		}
	}()
	return nil
}

// DownloadFileWithCallback 同步下载文件，通过 progressFn(written, total) 实时回传进度。
// ctx 用于支持取消；调用方应在 goroutine 中运行此方法以避免阻塞。
func (f FileOp) DownloadFileWithCallback(ctx context.Context, url, dst string, ignoreCertificate bool, progressFn func(written, total uint64)) error {
	client := &http.Client{}
	if ignoreCertificate {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept-Encoding", "identity")
	resp, err := client.Do(request)
	if err != nil {
		// 如果是取消操作，返回明确错误
		if errors.Is(err, context.Canceled) {
			return err
		}
		global.LOG.Errorf("download file [%s] error, err %s", dst, err.Error())
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dst)
	if err != nil {
		global.LOG.Errorf("create download file [%s] error, err %s", dst, err.Error())
		return err
	}
	defer out.Close()

	var total uint64
	if resp.ContentLength > 0 {
		total = uint64(resp.ContentLength)
	}

	progressReader := &progressCallbackReader{
		ctx:        ctx,
		reader:     resp.Body,
		written:    0,
		total:      total,
		progressFn: progressFn,
	}

	if _, err = io.Copy(out, progressReader); err != nil {
		if errors.Is(err, context.Canceled) {
			_ = os.Remove(dst)
			return err
		}
		return fmt.Errorf("save download file [%s] error, err %s", dst, err.Error())
	}
	return nil
}

// progressCallbackReader 包装 io.Reader，支持取消，每次读取后回调进度
type progressCallbackReader struct {
	ctx        context.Context
	reader     io.Reader
	written    uint64
	total      uint64
	progressFn func(written, total uint64)
}

func (p *progressCallbackReader) Read(buf []byte) (int, error) {
	// 检查是否已取消
	select {
	case <-p.ctx.Done():
		return 0, p.ctx.Err()
	default:
	}
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.written += uint64(n)
		if p.progressFn != nil {
			p.progressFn(p.written, p.total)
		}
	}
	return n, err
}

func (f FileOp) DownloadFile(url, dst string) error {
	resp, err := http2.GetHttpRes(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create download file [%s] error, err %s", dst, err.Error())
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("save download file [%s] error, err %s", dst, err.Error())
	}
	return nil
}

func (f FileOp) DownloadFileWithProxy(url, dst string) error {
	_, resp, err := http2.HandleGet(url, http.MethodGet, constant.TimeOut5m)
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create download file [%s] error, err %s", dst, err.Error())
	}
	defer out.Close()

	reader := bytes.NewReader(resp)
	if _, err = io.Copy(out, reader); err != nil {
		return fmt.Errorf("save download file [%s] error, err %s", dst, err.Error())
	}
	return nil
}
