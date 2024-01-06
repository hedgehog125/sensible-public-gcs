package subfns

import (
	"fmt"
	"sync"
	"time"

	"github.com/hedgeghog125/sensible-public-gcs/endpoints"
	"github.com/hedgeghog125/sensible-public-gcs/intertypes"
)

func UsersTick(state *intertypes.State, env *intertypes.Env) {
	var wg sync.WaitGroup

	now := time.Now()
	for ip, userChan := range state.Users {
		wg.Add(1)
		ip, userChan := ip, userChan

		go func() {
			defer wg.Done()
			user := <-*userChan

			if endpoints.UserTick(user, now, env) {
				fmt.Printf("Forgot %v\n", ip)
				delete(state.Users, ip)
				user = nil // So nil is put back into the channel
			}

			go func() { *userChan <- user }()
		}()
	}

	wg.Wait()
}
