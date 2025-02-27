package underhood

import (
	"fmt"

	"github.com/henrycg/simplepir/matrix"
	"github.com/henrycg/simplepir/pir"
	"github.com/henrycg/simplepir/rand"
)

type KeyBlob = []byte

type CipherBlob = []byte

type HintQuery = []CipherBlob

// JAY:Client query without tokens
type QueryWithEncSecrets[T matrix.Elem] struct {
	pirQuery       *pir.Query[T]
	encOuterSecret HintQuery
}

func (qu *QueryWithEncSecrets[T]) printSize() {
	qu_bytes := (len(qu.pirQuery.Query.Data()) * int(T(0).Bitlen())) / 8

	hint_bytes := int(0)
	for i := 0; i < len(qu.encOuterSecret); i++ {
		hint_bytes += len(qu.encOuterSecret[i])
	}

	fmt.Printf("(Client query) PIR query size=%d bytes; Outer Enc size=%d bytes; Total size=%d bytes\n", qu_bytes, hint_bytes, qu_bytes+hint_bytes)
}

type HintAnswer struct {
	MatrixRows uint64
	HintCts    [][]CipherBlob
}

type Client[T matrix.Elem] struct {
	params    *params
	pirClient *pir.Client[T]

	innerSecret *matrix.Matrix[T]
	outerSecret KeyBlob

	interm *matrix.Matrix[T]
	skLHE  *pir.SecretLHE[T]
	sk     *pir.Secret[T]
}

// WARNING: You must call Free() on this client to cleanup
func NewClient[T matrix.Elem](matrixAseed *rand.PRGKey, dbinfo *pir.DBInfo) *Client[T] {
	return &Client[T]{
		params:    newParams(),
		pirClient: pir.NewClient[T](nil, matrixAseed, dbinfo),
	}
}

// WARNING: You must call Free() on this client to cleanup
func NewClientDistributed[T matrix.Elem](matrixAseeds []rand.PRGKey, offsets []uint64, dbinfo *pir.DBInfo) *Client[T] {
	return &Client[T]{
		params:    newParams(),
		pirClient: pir.NewClientDistributed[T](nil, matrixAseeds, offsets, dbinfo),
	}
}

func (c *Client[T]) Free() {
	c.params.Free()
}

func (c *Client[T]) QueryWithEncSecrets(q uint64) *QueryWithEncSecrets[T] {
	var encSk HintQuery

	c.innerSecret = c.pirClient.GenerateSecret()
	c.outerSecret, encSk = c.encryptSecret(c.innerSecret)

	c.sk = c.pirClient.PreprocessQueryGivenSecret(c.innerSecret)
	query := c.pirClient.QueryPreprocessed(q, c.sk)

	return &QueryWithEncSecrets[T]{
		pirQuery:       query,
		encOuterSecret: encSk,
	}
}

func (c *Client[T]) HintQuery() *HintQuery {
	var encSk HintQuery

	c.innerSecret = c.pirClient.GenerateSecret()
	c.outerSecret, encSk = c.encryptSecret(c.innerSecret)

	return &encSk
}

func (c *Client[T]) CopySecret(oc *Client[matrix.Elem64]) {
	switch v := any(c).(type) {
	case *Client[matrix.Elem64]:
		v.innerSecret = oc.innerSecret.Copy()

	case *Client[matrix.Elem32]:
		if oc.pirClient.GetSecurityParam() < v.pirClient.GetSecurityParam() {
			panic("Invalid operation")
		}
		toDrop := oc.pirClient.GetSecurityParam() - v.pirClient.GetSecurityParam()

		smallSecret := oc.innerSecret.Make32()
		smallSecret.DropLastrows(toDrop)
		v.innerSecret = smallSecret

	default:
		panic("Should not get here")
	}

	c.outerSecret = make([]byte, len(oc.outerSecret))
	copy(c.outerSecret, oc.outerSecret)
}

func (c *Client[T]) PreprocessQuery() {
	c.sk = c.pirClient.PreprocessQueryGivenSecret(c.innerSecret)
}

func (c *Client[T]) PreprocessQueryLHE() {
	c.skLHE = c.pirClient.PreprocessQueryLHEGivenSecret(c.innerSecret)
}

// Recover H.s
func (c *Client[T]) HintRecover(ans *HintAnswer) {
	c.interm = c.recoverAS(ans)
}

func (c *Client[T]) Query(q uint64) *pir.Query[T] {
	return c.pirClient.QueryPreprocessed(q, c.sk)
}

func (c *Client[T]) QueryLHE(q *matrix.Matrix[T]) *pir.Query[T] {
	return c.pirClient.QueryLHEPreprocessed(q, c.skLHE)
}

func (c *Client[T]) Recover(ansIn *pir.Answer[T]) []uint64 {
	ans := ansIn.Answer.Copy()
	ans.Sub(c.interm)
	return c.pirClient.DecodeMany(ans)
}

func (c *Client[T]) RecoverAnswerWithHintCts(ansIn *ServerResponseWithHintAnswer[T]) []uint64 {
	// decrypt hint cts and recover part (H=(Db x A) x s)
	as := c.recoverAS(&ansIn.hintAns)

	ans := ansIn.pirAnswer.Answer.Copy()

	// (Db x (As + e + \delta(q))) - (H=(Db x A) x s)
	ans.Sub(as)
	return c.pirClient.DecodeMany(ans)
}

func (c *Client[T]) RecoverLHE(ansIn *pir.Answer[T]) *matrix.Matrix[T] {
	ans := ansIn.Answer.Copy()
	ans.Sub(c.interm)
	return c.pirClient.DecodeManyLHE(ans)
}
