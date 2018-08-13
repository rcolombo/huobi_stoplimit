package main

import "fmt"

func (h *Huobi) OrdersEqual(old, new [][]float64) bool {
	if len(old) != len(new) {
		return false
	} else {
		for i, v := range new {
			if !(v[0] == old[i][0] && v[1] == old[i][1]) {
				return false
			}
		}
	}
	return true
}

func (h *Huobi) GetDepthChanges(old, new [][]float64) ([][]float64, [][]float64) {
	var removed, added [][]float64
	if h.OrdersEqual(old, new) {
		return removed, added
	}

	newKeys := map[string][]float64{}
	for _, o := range new {
		newKeys[h.OrderKey(o)] = o
	}
	for _, o := range old {
		if _, ok := newKeys[h.OrderKey(o)]; !ok {
			// this price does not exist in the new update, so treat it as a deletion
			removed = append(removed, o)
		} else if newKeys[h.OrderKey(o)][1] == o[1] {
			// this price exists in the new update unchanged, so delete it from the new updates
			delete(newKeys, h.OrderKey(o))
		}
	}
	for _, order := range newKeys {
		added = append(added, order)
	}
	return removed, added
}

func (h *Huobi) OrderKey(order []float64) string {
	price := order[0]
	return fmt.Sprintf("%.10f", price)
}
