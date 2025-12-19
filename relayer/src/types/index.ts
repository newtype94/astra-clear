import { BigNumber } from 'ethers';

/**
 * Transfer event emitted from Besu Gateway contract
 * Requirement 1.2: 소각 이벤트 모니터링
 */
export interface TransferEvent {
  txHash: string;
  blockNumber: number;
  sender: string;
  recipient: string;
  amount: BigNumber;
  sourceChain: string;
  destChain: string;
  nonce: BigNumber;
  timestamp: number;
}

/**
 * Mint command from Cosmos Hub multisig module
 * Requirement 5.4: 발행 명령 실행
 */
export interface MintCommand {
  commandId: string;
  targetChain: string;
  recipient: string;
  amount: BigNumber;
  signatures: ECDSASignature[];
  createdAt: number;
  status: MintCommandStatus;
}

/**
 * ECDSA signature for cross-chain commands
 * Requirement 5.5: 서명 검증
 */
export interface ECDSASignature {
  validator: string;
  r: string;
  s: string;
  v: number;
}

export enum MintCommandStatus {
  Pending = 'pending',
  Signed = 'signed',
  Executed = 'executed',
  Failed = 'failed',
}

/**
 * Configuration for the relayer service
 */
export interface RelayerConfig {
  // Besu configuration
  besuRpcUrl: string;
  besuWsUrl: string;
  gatewayAddress: string;
  executorAddress: string;
  networkId: string;

  // Cosmos configuration
  cosmosRpcUrl: string;
  cosmosChainId: string;
  cosmosPrefix: string;

  // Relayer configuration
  pollInterval: number; // milliseconds
  maxRetries: number;
  retryDelay: number; // milliseconds
  startBlock?: number;

  // Keys
  relayerMnemonic: string;
  besuPrivateKey: string;
}

/**
 * Event log for tracking processed events
 */
export interface EventLog {
  txHash: string;
  blockNumber: number;
  eventType: 'transfer' | 'mint';
  processed: boolean;
  processedAt?: Date;
  error?: string;
}

/**
 * Vote message for Cosmos Hub oracle module
 * Requirement 3.1: 투표 제출
 */
export interface VoteMessage {
  validator: string;
  transferEvent: TransferEvent;
  signature: string;
}
