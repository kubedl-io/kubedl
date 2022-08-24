package util

import (
	"k8s.io/apimachinery/pkg/util/errors"
)

func NewAggregatedErrors() AggregatedErrors {
	return make(AggregatedErrors, 0)
}

type AggregatedErrors []error

func (agg *AggregatedErrors) Collect(err error) {
	if err == nil {
		return
	}
	*agg = append(*agg, err)
}

func (agg *AggregatedErrors) Errors() []error {
	return *agg
}

func (agg *AggregatedErrors) Error() error {
	return errors.NewAggregate(*agg)
}

func (agg *AggregatedErrors) Count() int {
	return len(*agg)
}
