package underhood

import (
	"testing"

	"github.com/henrycg/simplepir/lwe"
	"github.com/henrycg/simplepir/matrix"
	"github.com/henrycg/simplepir/pir"
	"github.com/henrycg/simplepir/rand"
)

func testTree[IntT matrix.Elem](t *testing.T, dbSize uint64) {
	pMod := uint64(512)
	seed := rand.RandomPRGKey() // matrix A seed
	params := lwe.NewParamsFixedP(IntT(0).Bitlen(), 1<<10, pMod)
	db := pir.NewDatabaseRandomFixedParams[IntT](rand.NewRandomBufPRG(), dbSize, 1, params)

	server := NewServer(db, seed)
	defer server.Free()

	client := NewClient[IntT](seed, db.Info)
	defer client.Free()

	// Create Query
	idx := uint64(7)
	query := client.QueryWithEncSecrets(idx)

	// Server
	ans := server.AnswerWithHintCts(query)

	// Recover response
	msg := client.RecoverAnswerWithHintCts(ans)

	for row := 0; row < len(msg); row++ {
		i := uint64(row)*db.Info.M + (idx % db.Info.M)
		if db.GetElem(i) != msg[row] {
			t.Fail()
		}
	}
}

func TestTreeHuge64(t *testing.T) {
	testTree[matrix.Elem64](t, 1<<24)
}
