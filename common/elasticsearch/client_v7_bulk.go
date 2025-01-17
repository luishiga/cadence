// The MIT License (MIT)

// Copyright (c) 2017-2020 Uber Technologies Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package elasticsearch

import (
	"context"

	"github.com/olivere/elastic/v7"
)

var _ GenericBulkProcessor = (*v7BulkProcessor)(nil)

type v7BulkProcessor struct {
	processor *elastic.BulkProcessor
}

func (c *elasticV7) RunBulkProcessor(ctx context.Context, parameters *BulkProcessorParameters) (GenericBulkProcessor, error) {
	beforeFunc := func(executionId int64, requests []elastic.BulkableRequest) {
		parameters.BeforeFunc(executionId, fromV7ToGenericBulkableRequests(requests))
	}

	afterFunc := func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
		gerr := convertV7ErrorToGenericError(err)
		parameters.AfterFunc(
			executionId,
			fromV7ToGenericBulkableRequests(requests),
			fromV7toGenericBulkResponse(response),
			gerr)
	}

	processor, err := c.client.BulkProcessor().
		Name(parameters.Name).
		Workers(parameters.NumOfWorkers).
		BulkActions(parameters.BulkActions).
		BulkSize(parameters.BulkSize).
		FlushInterval(parameters.FlushInterval).
		Backoff(parameters.Backoff).
		Before(beforeFunc).
		After(afterFunc).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	return &v7BulkProcessor{
		processor: processor,
	}, nil
}

func (v *v7BulkProcessor) Flush() error {
	return v.processor.Flush()
}

func (v *v7BulkProcessor) Start(ctx context.Context) error {
	return v.processor.Start(ctx)
}

func (v *v7BulkProcessor) Stop() error {
	return v.processor.Stop()
}

func (v *v7BulkProcessor) Close() error {
	return v.processor.Close()
}

func (v *v7BulkProcessor) Add(request *GenericBulkableAddRequest) {
	var req elastic.BulkableRequest
	switch request.RequestType {
	case BulkableDeleteRequest:
		req = elastic.NewBulkDeleteRequest().
			Index(request.Index).
			Id(request.ID).
			VersionType(request.VersionType).
			Version(request.Version)
	case BulkableIndexRequest:
		req = elastic.NewBulkIndexRequest().
			Index(request.Index).
			Id(request.ID).
			VersionType(request.VersionType).
			Version(request.Version).
			Doc(request.Doc)
	case BulkableCreateRequest:
		//for bulk create request still calls the bulk index method
		//with providing operation type
		req = elastic.NewBulkIndexRequest().
			OpType("create").
			Index(request.Index).
			Id(request.ID).
			VersionType("internal").
			Doc(request.Doc)
	}
	v.processor.Add(req)
}

func convertV7ErrorToGenericError(err error) *GenericError {
	if err == nil {
		return nil
	}
	status := unknownStatusCode
	switch e := err.(type) {
	case *elastic.Error:
		status = e.Status
	}
	return &GenericError{
		Status:  status,
		Details: err,
	}
}

func fromV7toGenericBulkResponse(response *elastic.BulkResponse) *GenericBulkResponse {
	if response == nil {
		return &GenericBulkResponse{}
	}
	return &GenericBulkResponse{
		Took:   response.Took,
		Errors: response.Errors,
		Items:  fromV7ToGenericBulkResponseItemMaps(response.Items),
	}
}

func fromV7ToGenericBulkResponseItemMaps(items []map[string]*elastic.BulkResponseItem) []map[string]*GenericBulkResponseItem {
	var gitems []map[string]*GenericBulkResponseItem
	for _, it := range items {
		gitems = append(gitems, fromV7ToGenericBulkResponseItemMap(it))
	}
	return gitems
}

func fromV7ToGenericBulkResponseItemMap(m map[string]*elastic.BulkResponseItem) map[string]*GenericBulkResponseItem {
	if m == nil {
		return nil
	}
	gm := make(map[string]*GenericBulkResponseItem, len(m))
	for k, v := range m {
		gm[k] = fromV7ToGenericBulkResponseItem(v)
	}
	return gm
}

func fromV7ToGenericBulkResponseItem(v *elastic.BulkResponseItem) *GenericBulkResponseItem {
	return &GenericBulkResponseItem{
		Index:         v.Index,
		Type:          v.Type,
		ID:            v.Id,
		Version:       v.Version,
		Result:        v.Result,
		SeqNo:         v.SeqNo,
		PrimaryTerm:   v.PrimaryTerm,
		Status:        v.Status,
		ForcedRefresh: v.ForcedRefresh,
	}
}

func fromV7ToGenericBulkableRequests(requests []elastic.BulkableRequest) []GenericBulkableRequest {
	var v7Reqs []GenericBulkableRequest
	for _, req := range requests {
		v7Reqs = append(v7Reqs, req)
	}
	return v7Reqs
}
