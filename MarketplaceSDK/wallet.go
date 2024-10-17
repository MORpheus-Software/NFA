// wallet.go
package marketplacesdk

// import (
// 	"context"
// 	"net/http"
// )

// // GetWallet retrieves the wallet address.
// func (c *Client) GetWallet(ctx context.Context) (*WalletResponse, error) {
//     var resp WalletResponse
//     err := c.request(ctx, http.MethodGet, "/wallet", nil, &resp)
//     if err != nil {
//         return nil, err
//     }
//     return &resp, nil
// }

// // SetupWalletWithPrivateKey sets up the wallet using a private key.
// func (c *Client) SetupWalletWithPrivateKey(ctx context.Context, privateKey string) (*WalletResponse, error) {
//     req := WalletRequest{PrivateKey: privateKey}
//     var resp WalletResponse
//     err := c.request(ctx, http.MethodPost, "/wallet/privateKey", req, &resp)
//     if err != nil {
//         return nil, err
//     }
//     return &resp, nil
// }

// // DeleteWallet removes the wallet from proxy storage.
// func (c *Client) DeleteWallet(ctx context.Context) (*StatusResponse, error) {
//     var resp StatusResponse
//     err := c.request(ctx, http.MethodDelete, "/wallet", nil, &resp)
//     if err != nil {
//         return nil, err
//     }
//     return &resp, nil
// }
