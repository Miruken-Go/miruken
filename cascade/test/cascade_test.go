package test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/miruken-go/miruken/api"
	"github.com/miruken-go/miruken/cascade"
	context2 "github.com/miruken-go/miruken/context"
	"github.com/miruken-go/miruken/handles"
	"github.com/miruken-go/miruken/promise"
	"github.com/miruken-go/miruken/provides"
	"github.com/miruken-go/miruken/setup"
	"github.com/stretchr/testify/suite"
)

type (
	Reservation struct {
		Id       uuid.UUID
		FlightNo string
	}

	SeatReservation struct {
		ReservationId uuid.UUID
		FlightNo      string
		SeatId        uuid.UUID
	}

	ReserveSeats struct {
		FlightNo string
		Count    int
	}

	SeatReserved struct {
		ReservationId uuid.UUID
		FlightNo      string
		SeatId        uuid.UUID
	}

	ChangeReservedSeat struct {
		ReservationId uuid.UUID
		OldSeatId     uuid.UUID
		NewSeatId     uuid.UUID
	}

	ReservedSeatChanged struct {
		ReservationId uuid.UUID
		OldSeatId     uuid.UUID
		NewSeatId     uuid.UUID
	}

	ReservationHandler struct {
		res map[uuid.UUID]*Reservation
	}

	SeatInventory struct {
		seats map[uuid.UUID][]SeatReservation
	}
)

func (r *ReservationHandler) Constructor() {
	r.res = make(map[uuid.UUID]*Reservation)
}

func (r *ReservationHandler) Reserve(
	_ *handles.It, reserve ReserveSeats,
) (Reservation, *cascade.Messages) {
	reservationId := uuid.New()
	reservation := Reservation{
		Id:       reservationId,
		FlightNo: reserve.FlightNo,
	}
	r.res[reservationId] = &reservation
	seats := make([]any, reserve.Count)
	for i := 0; i < reserve.Count; i++ {
		seats[i] = SeatReserved{
			ReservationId: reservationId,
			FlightNo:      reserve.FlightNo,
			SeatId:        uuid.New(),
		}
	}
	return reservation, cascade.Post(seats...)
}

func (r *ReservationHandler) ChangeReservedSeat(
	_ *handles.It, change ChangeReservedSeat,
) (*promise.Promise[struct{}], *cascade.Messages) {
	if _, ok := r.res[change.ReservationId]; !ok {
		return promise.Reject[struct{}](
			errors.New("reservation not found")), nil
	}
	return promise.Empty(),
		cascade.Publish(ReservedSeatChanged{
			ReservationId: change.ReservationId,
			OldSeatId:     change.OldSeatId,
			NewSeatId:     change.NewSeatId,
		})
}

func (s *SeatInventory) Constructor() {
	s.seats = make(map[uuid.UUID][]SeatReservation)
}

func (s *SeatInventory) Get(
	reservationId uuid.UUID,
) ([]SeatReservation, error) {
	if flight, ok := s.seats[reservationId]; ok {
		return flight, nil
	}
	return nil, errors.New("flight not found")
}

func (s *SeatInventory) Reserved(
	_ *handles.It, reserved SeatReserved,
) error {
	resId := reserved.ReservationId
	if seats, ok := s.seats[resId]; ok {
		s.seats[resId] = append(seats,
			SeatReservation{resId, reserved.FlightNo, reserved.SeatId})
	} else {
		s.seats[resId] = []SeatReservation{
			{resId, reserved.FlightNo, reserved.SeatId}}
	}
	return nil
}

func (s *SeatInventory) Changed(
	_ *handles.It, changed ReservedSeatChanged,
) *promise.Promise[struct{}] {
	resId := changed.ReservationId
	if seats, ok := s.seats[resId]; ok {
		for i, s := range seats {
			if s.SeatId == changed.OldSeatId {
				seats[i].SeatId = changed.NewSeatId
				break
			}
		}
	}
	return promise.Empty()
}

type CascadeTestSuite struct {
	suite.Suite
	specs []any
}

func (suite *CascadeTestSuite) SetupTest() {
	suite.specs = []any{
		&ReservationHandler{},
		&SeatInventory{},
	}
}

func (suite *CascadeTestSuite) Setup(specs ...any) (*context2.Context, error) {
	if len(specs) == 0 {
		specs = suite.specs
	}
	return setup.New().Specs(specs...).Context()
}

func (suite *CascadeTestSuite) TestCascade() {
	suite.Run("Post", func() {
		handler, _ := suite.Setup()
		reservation, _, err := api.Send[Reservation](handler, ReserveSeats{
			FlightNo: "UA 123",
			Count:    5,
		})
		suite.Nil(err)
		suite.NotNil(reservation)

		inventory, _, ok, err := provides.Type[*SeatInventory](handler)
		suite.True(ok)
		suite.Nil(err)
		suite.NotNil(inventory)
		seats, err := inventory.Get(reservation.Id)
		suite.Nil(err)
		suite.Len(seats, 5)

		for _, seat := range seats {
			suite.Equal("UA 123", seat.FlightNo)
			suite.Equal(reservation.Id, seat.ReservationId)
		}
	})

	suite.Run("Publish", func() {
		handler, _ := suite.Setup()
		reservation, _, err := api.Send[Reservation](handler,
			ReserveSeats{
				FlightNo: "AA 139",
				Count:    3,
			})
		suite.Nil(err)
		suite.NotNil(reservation)

		inventory, _, _, err := provides.Type[*SeatInventory](handler)
		seats, err := inventory.Get(reservation.Id)

		newSeatId := uuid.New()
		pc, err := api.Post(handler,
			ChangeReservedSeat{
				ReservationId: reservation.Id,
				OldSeatId:     seats[0].SeatId,
				NewSeatId:     newSeatId,
			})
		suite.Nil(err)
		suite.NotNil(pc)
		_, err = pc.Await()
		suite.Nil(err)

		seats, err = inventory.Get(reservation.Id)
		suite.Nil(err)
		suite.Equal(newSeatId, seats[0].SeatId)
	})
}

func TestCascadeTestSuite(t *testing.T) {
	suite.Run(t, new(CascadeTestSuite))
}
