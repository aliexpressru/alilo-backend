package data

import (
	"math"
)

// OffsetCalculation Функция рассчитывает offset для запроса пагинации из "ограничения вывода" и "номера страницы"
// функция не учитывает фактическое наличие данных в таблице
func OffsetCalculation(limit int32, pageNumber int32) (offset int32, returnLimit int32) {
	offset = int32(0)

	if limit < 1 {
		limit = 10
	}

	if pageNumber > 1 {
		offset = limit * (pageNumber - 1)
	}

	return offset, limit
}

// CalculateTotalPages Функция рассчитывает кол-во строк для страницы исходя из ограничения(limit) и общего кол-ва строк(count)//todo: добавить примеры
func CalculateTotalPages(numberLines int64, limit int32) (totalPages int64) {
	return int64(math.Ceil(float64(numberLines) / float64(limit)))
}
