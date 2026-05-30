package protocol

import (
	"errors"
	"fmt"
	"time"
)

type MinuteResp struct {
	Count uint16
	List  []PriceNumber
}

type PriceNumber struct {
	Time   string
	Price  Price
	Avg    Price
	Number int
}

func (this PriceNumber) String() string {
	return fmt.Sprintf("%s \t%-6s \t%-6s \t%-6d(手)", this.Time, this.Price, this.Avg, this.Number)
}

type minute struct{}

func (this *minute) Frame(code string) (*Frame, error) {
	exchange, number, err := DecodeCode(code)
	if err != nil {
		return nil, err
	}
	codeBs := []byte(number)
	codeBs = append(codeBs, 0x0, 0x0, 0x0, 0x0)
	return &Frame{
		Control: Control01,
		Type:    TypeMinute,
		Data:    append([]byte{exchange.Uint8(), 0x0}, codeBs...),
	}, nil
}

func (this *minute) Decode(bs []byte) (*MinuteResp, error) {

	if len(bs) < 4 {
		return nil, errors.New("数据长度不足")
	}

	resp := &MinuteResp{
		Count: Uint16(bs[:2]),
	}
	bs = bs[4:]

	startPrice := Price(0)
	startAvg := Price(0)

	t := time.Date(0, 0, 0, 9, 30, 0, 0, time.Local)
	for i := uint16(0); i < resp.Count; i++ {
		var price Price
		bs, price = GetPrice(bs)

		var avg Price
		bs, avg = GetPrice(bs)

		var vol int
		bs, vol = CutInt(bs)

		if i > 0 {
			price += startPrice
			avg += startAvg
		} else {
			startPrice = price
			startAvg = avg
		}

		if i == 120 {
			t = t.Add(time.Minute * 90)
		}

		resp.List = append(resp.List, PriceNumber{
			Time:   t.Add(time.Minute * time.Duration(i+1)).Format("15:04"),
			Price:  price * 10,
			Avg:    avg / 10,
			Number: vol,
		})
	}

	return resp, nil
}
