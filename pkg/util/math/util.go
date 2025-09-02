package math

import (
	"context"
	crand "crypto/rand"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/aliexpressru/alilo-backend/pkg/util/logger"
)

// GeometricRandomValue - Генерация геометрически распределенного случайного числа
func GeometricRandomValue(ctx context.Context, p float64, n int) int {
	// p - параметр геометрического распределения
	// N - ограничиваем результат верхней границей
	randomSource := int(math.Floor(math.Log(1-Float64(ctx)) / math.Log(1-p)))
	// Масштабируем результат в заданный диапазон
	randomNumber := randomSource % n

	return randomNumber
}

// Intn is a shortcut for generating a random integer between 0 and
// max using crypto/rand.
func Intn(ctx context.Context, n float64) int64 {
	nBig, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		logger.Errorf(ctx, "", err)
	}

	return nBig.Int64()
}
func Float64(ctx context.Context) float64 {
	return float64(Intn(ctx, 1<<53)) / (1 << 53)
}

func GetRandomID32(ctx context.Context) int32 {
	id := randomInt(ctx, 32)
	logger.Infof(ctx, "GetID: installed: %v", id)

	//nolint:gosec // id всегда в пределах int32
	return int32(id)
}

func randomInt(ctx context.Context, bitSize int) int64 {
	count, err := strconv.ParseInt(strings.Repeat("9", 8), 10, bitSize)
	if err != nil {
		logger.Error(ctx, "RandomInt error: ", err)
	}

	logger.Infof(ctx, "RandomInt count: '%v'; ", count)

	nBig, err := crand.Int(crand.Reader, big.NewInt(count))
	if err != nil {
		logger.Error(ctx, "RandomInt error: ", err)
	}

	logger.Infof(ctx, "RandomInt: %v", nBig)

	return nBig.Int64()
}

func Int32Fm(str string) int32 {
	//nolint:gosec // str всегда в пределах int32
	return int32(int64Fm(str, 32))
}

func Int64Fm(str string) int64 {
	return int64Fm(str, 64)
}

func int64Fm(str string, bitSize int) int64 {
	i, _ := strconv.ParseInt(str, 10, bitSize)

	return i
}
