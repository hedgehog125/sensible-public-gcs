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

	now := time.Now().UTC()
	state.Users.Range(func(ip string, userChan chan *intertypes.User) bool {
		wg.Add(1)

		go func() {
			defer wg.Done()
			user := <-userChan

			if endpoints.UserTick(user, now, env) {
				if !env.DISABLE_REQUEST_LOGS {
					fmt.Printf("forgot %v\n", ip)
				}
				state.Users.Delete(ip)
				user = nil // So nil is put back into the channel
			}

			go func() { userChan <- user }()
		}()
		return true
	})
	wg.Wait()
}
