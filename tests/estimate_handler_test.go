package tests

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"testing"

	"bigswapenergy/internal/presentation/http"
	"bigswapenergy/internal/shared/config"
	usecases "bigswapenergy/internal/usecases"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type mockEstimateService struct {
	estimateAmount *big.Int
	estimateError  error
}

func (m *mockEstimateService) EstimateSwapAmount(ctx context.Context, poolAddress, srcToken, dstToken string, srcAmount *big.Int) (*big.Int, error) {
	return m.estimateAmount, m.estimateError
}

func createEstimateHandler(estimateService usecases.EstimateService) *http.EstimateHandler {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{
		RateLimit: config.RateLimitConfig{
			RequestsPerMinute: 100,
		},
	}
	return http.NewEstimateHandler(estimateService, logger, cfg)
}

func TestEstimateSwapAmount_Success(t *testing.T) {
	mockService := &mockEstimateService{
		estimateAmount: big.NewInt(996),
		estimateError:  nil,
	}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=1000")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusOK, ctx.Response.StatusCode())
	}

	expectedBody := "996"
	actualBody := string(ctx.Response.Body())
	if actualBody != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, actualBody)
	}

	contentType := string(ctx.Response.Header.ContentType())
	if contentType != "text/plain" {
		t.Errorf("Expected content type text/plain, got %s", contentType)
	}
}

func TestEstimateSwapAmount_MissingPool(t *testing.T) {
	mockService := &mockEstimateService{}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?src=0x456&dst=0x789&src_amount=1000")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_MissingSrc(t *testing.T) {
	mockService := &mockEstimateService{}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&dst=0x789&src_amount=1000")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_MissingDst(t *testing.T) {
	mockService := &mockEstimateService{}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&src_amount=1000")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_MissingSrcAmount(t *testing.T) {
	mockService := &mockEstimateService{}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_InvalidSrcAmount(t *testing.T) {
	mockService := &mockEstimateService{}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=invalid")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_ZeroSrcAmount(t *testing.T) {
	mockService := &mockEstimateService{}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=0")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_NegativeSrcAmount(t *testing.T) {
	mockService := &mockEstimateService{}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=-100")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusBadRequest, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_LargeNumbers(t *testing.T) {
	mockService := &mockEstimateService{
		estimateAmount: big.NewInt(999999999999),
		estimateError:  nil,
	}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=1000000000000")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusOK, ctx.Response.StatusCode())
	}

	expectedBody := "999999999999"
	actualBody := string(ctx.Response.Body())
	if actualBody != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, actualBody)
	}
}

func TestEstimateSwapAmount_ServiceError(t *testing.T) {
	mockService := &mockEstimateService{
		estimateAmount: nil,
		estimateError:  fmt.Errorf("pool not found"),
	}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=1000")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
	}
}

func TestEstimateSwapAmount_URLEncoding(t *testing.T) {
	mockService := &mockEstimateService{
		estimateAmount: big.NewInt(996),
		estimateError:  nil,
	}
	handler := createEstimateHandler(mockService)

	poolAddr := "0x1234567890123456789012345678901234567890"
	srcToken := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	dstToken := "0xfedcbafedcbafedcbafedcbafedcbafedcbafedc"

	params := url.Values{}
	params.Set("pool", poolAddr)
	params.Set("src", srcToken)
	params.Set("dst", dstToken)
	params.Set("src_amount", "1000")

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?" + params.Encode())
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(req, nil, nil)

	handler.EstimateSwapAmount(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status %d, got %d", fasthttp.StatusOK, ctx.Response.StatusCode())
	}

	expectedBody := "996"
	actualBody := string(ctx.Response.Body())
	if actualBody != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, actualBody)
	}
}

func TestEstimateSwapAmount_EdgeCases(t *testing.T) {
	testCases := []struct {
		name           string
		srcAmount      string
		expectedStatus int
	}{
		{"minimum_valid", "1", fasthttp.StatusOK},
		{"large_number", "999999999999999999", fasthttp.StatusOK},
		{"zero", "0", fasthttp.StatusBadRequest},
		{"negative", "-1", fasthttp.StatusBadRequest},
		{"float", "100.5", fasthttp.StatusBadRequest},
		{"empty", "", fasthttp.StatusBadRequest},
		{"non_numeric", "abc", fasthttp.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := &mockEstimateService{
				estimateAmount: big.NewInt(996),
				estimateError:  nil,
			}
			handler := createEstimateHandler(mockService)

			req := fasthttp.AcquireRequest()
			defer fasthttp.ReleaseRequest(req)

			req.SetRequestURI(fmt.Sprintf("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=%s", tc.srcAmount))
			req.Header.SetMethod("GET")

			ctx := &fasthttp.RequestCtx{}
			ctx.Init(req, nil, nil)

			handler.EstimateSwapAmount(ctx)

			if ctx.Response.StatusCode() != tc.expectedStatus {
				t.Errorf("Test case %s: expected status %d, got %d", tc.name, tc.expectedStatus, ctx.Response.StatusCode())
			}
		})
	}
}

func BenchmarkEstimateSwapAmount(b *testing.B) {
	mockService := &mockEstimateService{
		estimateAmount: big.NewInt(996),
		estimateError:  nil,
	}
	handler := createEstimateHandler(mockService)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI("/estimate?pool=0x123&src=0x456&dst=0x789&src_amount=1000")
	req.Header.SetMethod("GET")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &fasthttp.RequestCtx{}
		ctx.Init(req, nil, nil)
		handler.EstimateSwapAmount(ctx)
	}
}
