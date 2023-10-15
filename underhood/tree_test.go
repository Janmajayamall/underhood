package underhood

import (
	"fmt"
	"testing"

	"github.com/henrycg/simplepir/lwe"
	"github.com/henrycg/simplepir/matrix"
	"github.com/henrycg/simplepir/pir"
	"github.com/henrycg/simplepir/rand"
)

// NOTE: These parameters were chosen to support ternary secrets

// For 32-bit ciphertext modulus
const secretDimension32 = uint64(1408)
const lweErrorStdDev32 = float64(6.4)

// For 64-bit ciphertext modulus
const secretDimension64 = uint64(2048)
const lweErrorStdDev64 = float64(81920.0)

/* Maps #samples ==> plaintext modulus */
var plaintextModulus32 = map[uint64]uint64{
	1 << 13: 991,
	1 << 14: 833,
	1 << 15: 701,
	1 << 16: 589,
	1 << 17: 495,
	1 << 18: 416,
	1 << 19: 350,
	1 << 20: 294,
}

/* Maps #samples ==> plaintext modulus */
var plaintextModulus64 = map[uint64]uint64{
	1 << 13: 574457,
	1 << 14: 483058,
	1 << 15: 406202,
	1 << 16: 341574,
	1 << 17: 287228,
	1 << 18: 241529,
	1 << 19: 203101,
	1 << 20: 170787,
	1 << 21: 143614,
	//1 << 22: 120764,
	//1 << 23: 101550,
	//1 << 24: 85393,
	//1 << 25: 71807,
	//1 << 26: 60382,
	//1 << 27: 50775,
}

func testTree[IntT matrix.Elem](t *testing.T, db_size uint64, db_m uint64, pt uint64, logpt uint64, query_repeat int) {
	seed := rand.RandomPRGKey() // matrix A seed
	params := lwe.NewParamsFixedP(IntT(0).Bitlen(), db_m, pt)
	db := pir.NewDatabaseRandomFixedParams[IntT](rand.NewRandomBufPRG(), db_size, logpt, params)

	fmt.Printf("Db dimensions=%dx%d (LxM); Total db entries=%d; Bits per entry=%d\n",
		db.Info.L, db.Info.M, db.Info.Num, db.Info.RowLength)
	db.Info.Params.PrintParams()

	server := NewServer(db, seed)
	defer server.Free()

	for k := 0; k < query_repeat; k++ {
		fmt.Println()
		fmt.Printf("Iterartion %d\n", k)
		client := NewClient[IntT](seed, db.Info)
		defer client.Free()

		// Create Query
		idx := uint64(7)
		query := client.QueryWithEncSecrets(idx)
		query.printSize()

		// Server
		ans := server.AnswerWithHintCts(query)
		ans.printSize()

		// Recover response
		msg := client.RecoverAnswerWithHintCts(ans)

		for row := 0; row < len(msg); row++ {
			i := uint64(row)*db.Info.M + (idx % db.Info.M)
			if db.GetElem(i) != msg[row] {
				fmt.Println("Fail!")
				t.Fail()
			}
		}

		fmt.Printf("Iterartion %d succeeded!\n", k)
		fmt.Println()
	}
}

func TestTree32(t *testing.T) {
	testTree[matrix.Elem32](t, 1<<30, 1<<16, 1<<9, 9, 1)
}
