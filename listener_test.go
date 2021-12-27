package haijun_net

import (
	"log"
	"net/http"
	"testing"
)

//
//
//
//        ***************************     ***************************         *********      ************************
//      *****************************    ******************************      *********      *************************
//     *****************************     *******************************     *********     *************************
//    *********                         *********                *******    *********     *********
//    ********                          *********               ********    *********     ********
//   ********     ******************   *********  *********************    *********     *********
//   ********     *****************    *********  ********************     *********     ********
//  ********      ****************    *********     ****************      *********     *********
//  ********                          *********      ********             *********     ********
// *********                         *********         ******            *********     *********
// ******************************    *********          *******          *********     *************************
//  ****************************    *********            *******        *********      *************************
//    **************************    *********              ******       *********         *********************
//
//

func TestNewHjListener(t *testing.T) {
	type args struct {
		addr string
	}
	tests := []struct {
		name    string
		args    args
		want    Listener
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				addr: "127.0.0.1:8080",
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHjListener(tt.args.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHjListener() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			http.DefaultServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				log.Println("request:", r.URL.Path)
				_, err := w.Write([]byte("hello world"))
				if err != nil {
					log.Println("write error:", err)
				}
			})

			log.Println("server start", tt.args.addr)
			http.Serve(got, http.DefaultServeMux)
		})
	}
}
