import {
  isConnected,
  getPublicKey,
  signTransaction,
  requestAccess,
} from "@stellar/freighter-api";

export class FreighterWalletService {
  async connect(): Promise<{ publicKey: string }> {
    const connected = await isConnected();
    if (!connected) {
      throw new Error(
        "Freighter extension is not installed. Install it from https://www.freighter.app/"
      );
    }
    await requestAccess();
    const publicKey = await getPublicKey();
    if (!publicKey) {
      throw new Error("Freighter did not return a public key");
    }
    return { publicKey };
  }

  async disconnect(): Promise<void> {
    // Freighter does not expose a disconnect API; clearing state client-side is sufficient.
  }

  async signTransaction(txXdr: string, networkPassphrase?: string): Promise<string> {
    const signed = await signTransaction(txXdr, {
      networkPassphrase,
    });
    if (!signed) {
      throw new Error("Freighter did not return a signed transaction");
    }
    return signed;
  }
}
