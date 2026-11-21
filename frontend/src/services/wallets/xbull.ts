/**
 * xBull Wallet adapter.
 *
 * xBull exposes a window.xBull object injected by its browser extension.
 * The API follows the SEP-7 / Stellar Web Authentication conventions and
 * provides connect() and signXDR() methods.
 */

interface XBullAPI {
  connect(): Promise<{ publicKey: string }>;
  signXDR(xdr: string, options?: { network?: string }): Promise<{ signedXDR: string }>;
}

function getXBull(): XBullAPI {
  const w = window as unknown as { xBull?: XBullAPI };
  if (!w.xBull) {
    throw new Error(
      "xBull extension is not installed. Install it from https://xbull.app/"
    );
  }
  return w.xBull;
}

export class XBullWalletService {
  async connect(): Promise<{ publicKey: string }> {
    const result = await getXBull().connect();
    if (!result?.publicKey) {
      throw new Error("xBull did not return a public key");
    }
    return { publicKey: result.publicKey };
  }

  async disconnect(): Promise<void> {
    // xBull does not expose a disconnect method.
  }

  async signTransaction(txXdr: string, network?: string): Promise<string> {
    const result = await getXBull().signXDR(txXdr, { network });
    if (!result?.signedXDR) {
      throw new Error("xBull did not return a signed transaction");
    }
    return result.signedXDR;
  }
}
