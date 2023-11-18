package subfns

import (
	"fmt"
	"sync"
	"time"

	"github.com/hedgeghog125/sensible-public-gcs/endpoints"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func UsersTick(state *intertypes.State) {
	var wg sync.WaitGroup

	now := time.Now().Unix()
	for ip, userChan := range state.Users {
		wg.Add(1)
		ip, userChan := ip, userChan

		go func() {
			defer wg.Done()
			user := <-*userChan

			if endpoints.UserTick(user, now) {
				fmt.Printf("Forgot %v\n", ip)
				delete(state.Users, ip)
				return
			}

			go func() { *userChan <- user }()
		}()
	}

	wg.Wait()
}
