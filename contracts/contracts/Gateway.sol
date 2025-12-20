// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./BankToken.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/**
 * @title Gateway
 * @dev Gateway contract for initiating cross-chain transfers
 * Burns tokens on the source chain and emits TransferInitiated event
 *
 * Requirement 1.1: Immediately burn specified amount from sender's account
 * Requirement 1.2: Emit TransferInitiated event with sender, recipient, amount, nonce
 */
contract Gateway is Ownable, ReentrancyGuard {
    BankToken public token;

    // Chain identifiers
    string public sourceChain;

    // Nonce for tracking transfers
    uint256 public nonce;

    // Mapping to track processed transfers
    mapping(bytes32 => bool) public processedTransfers;

    // Events
    /**
     * @dev Emitted when a cross-chain transfer is initiated
     * Requirement 1.2: Event includes sender, recipient, amount, nonce
     */
    event TransferInitiated(
        bytes32 indexed transferId,
        address indexed sender,
        string recipient,
        uint256 amount,
        uint256 nonce,
        string sourceChain,
        string destChain,
        uint256 blockHeight,
        uint256 timestamp
    );

    event GatewayPaused(address indexed by);
    event GatewayUnpaused(address indexed by);

    // State
    bool public paused;

    modifier whenNotPaused() {
        require(!paused, "Gateway: paused");
        _;
    }

    constructor(
        address _token,
        string memory _sourceChain,
        address initialOwner
    ) Ownable(initialOwner) {
        require(_token != address(0), "Gateway: token is zero address");
        require(bytes(_sourceChain).length > 0, "Gateway: empty source chain");

        token = BankToken(_token);
        sourceChain = _sourceChain;
        nonce = 0;
        paused = false;
    }

    /**
     * @dev Initiates a cross-chain transfer by burning tokens
     * Requirement 1.1: Burns tokens immediately from sender
     * Requirement 1.2: Emits TransferInitiated event
     *
     * @param recipient The recipient address on the destination chain
     * @param amount The amount to transfer
     * @param destChain The destination chain identifier
     * @return transferId The unique identifier for this transfer
     */
    function sendToChain(
        string calldata recipient,
        uint256 amount,
        string calldata destChain
    ) external nonReentrant whenNotPaused returns (bytes32 transferId) {
        require(amount > 0, "Gateway: amount must be greater than 0");
        require(bytes(recipient).length > 0, "Gateway: empty recipient");
        require(bytes(destChain).length > 0, "Gateway: empty destination chain");
        require(
            keccak256(bytes(destChain)) != keccak256(bytes(sourceChain)),
            "Gateway: cannot transfer to same chain"
        );

        // Check sender has sufficient balance
        require(
            token.balanceOf(msg.sender) >= amount,
            "Gateway: insufficient balance"
        );

        // Increment nonce
        nonce++;

        // Generate unique transfer ID
        transferId = keccak256(
            abi.encodePacked(
                msg.sender,
                recipient,
                amount,
                nonce,
                sourceChain,
                destChain,
                block.number,
                block.timestamp
            )
        );

        // Ensure transfer hasn't been processed before
        require(!processedTransfers[transferId], "Gateway: transfer already processed");
        processedTransfers[transferId] = true;

        // Requirement 1.1: Burn tokens immediately from sender
        token.burn(msg.sender, amount);

        // Requirement 1.2: Emit TransferInitiated event with all required data
        emit TransferInitiated(
            transferId,
            msg.sender,
            recipient,
            amount,
            nonce,
            sourceChain,
            destChain,
            block.number,
            block.timestamp
        );

        return transferId;
    }

    /**
     * @dev Pauses the gateway (emergency stop)
     */
    function pause() external onlyOwner {
        paused = true;
        emit GatewayPaused(msg.sender);
    }

    /**
     * @dev Unpauses the gateway
     */
    function unpause() external onlyOwner {
        paused = false;
        emit GatewayUnpaused(msg.sender);
    }

    /**
     * @dev Updates the token contract address
     * @param _token The new token contract address
     */
    function setToken(address _token) external onlyOwner {
        require(_token != address(0), "Gateway: token is zero address");
        token = BankToken(_token);
    }

    /**
     * @dev Returns the current nonce
     */
    function getCurrentNonce() external view returns (uint256) {
        return nonce;
    }

    /**
     * @dev Checks if a transfer has been processed
     * @param transferId The transfer ID to check
     */
    function isTransferProcessed(bytes32 transferId) external view returns (bool) {
        return processedTransfers[transferId];
    }
}
