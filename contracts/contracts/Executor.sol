// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./BankToken.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "@openzeppelin/contracts/utils/cryptography/MessageHashUtils.sol";

/**
 * @title Executor
 * @dev Executor contract for executing cross-chain mint commands
 * Verifies multi-signatures and mints tokens on the destination chain
 *
 * Requirement 1.3: Mint exact amount to recipient on destination chain
 * Requirement 5.4: Verify all signatures using ecrecover precompile
 * Requirement 5.5: Reject mint command if signature verification fails
 */
contract Executor is Ownable, ReentrancyGuard {
    using ECDSA for bytes32;
    using MessageHashUtils for bytes32;

    BankToken public token;

    // Chain identifier
    string public chainId;

    // Validator set management
    mapping(address => bool) public validators;
    address[] public validatorList;
    uint256 public threshold; // Required number of signatures (2/3 of validators)

    // Processed commands tracking
    mapping(bytes32 => bool) public processedCommands;

    // Validator set version for updates
    uint256 public validatorSetVersion;

    // Events
    event MintExecuted(
        bytes32 indexed commandId,
        address indexed recipient,
        uint256 amount,
        uint256 timestamp
    );

    event MintRejected(
        bytes32 indexed commandId,
        string reason
    );

    event ValidatorAdded(address indexed validator);
    event ValidatorRemoved(address indexed validator);
    event ValidatorSetUpdated(uint256 version, uint256 validatorCount, uint256 threshold);
    event ThresholdUpdated(uint256 oldThreshold, uint256 newThreshold);

    modifier onlyValidator() {
        require(validators[msg.sender], "Executor: caller is not a validator");
        _;
    }

    constructor(
        address _token,
        string memory _chainId,
        address initialOwner
    ) Ownable(initialOwner) {
        require(_token != address(0), "Executor: token is zero address");
        require(bytes(_chainId).length > 0, "Executor: empty chain id");

        token = BankToken(_token);
        chainId = _chainId;
        validatorSetVersion = 0;
        threshold = 1; // Default threshold, should be updated with validators
    }

    /**
     * @dev Executes a mint command after verifying signatures
     * Requirement 1.3: Mint exact amount to recipient
     * Requirement 5.4: Verify signatures using ecrecover
     * Requirement 5.5: Reject if signature verification fails
     *
     * @param commandId The unique command identifier
     * @param recipient The recipient address
     * @param amount The amount to mint
     * @param signatures Array of validator signatures
     */
    function executeMint(
        bytes32 commandId,
        address recipient,
        uint256 amount,
        bytes[] calldata signatures
    ) external nonReentrant {
        // Validation
        require(recipient != address(0), "Executor: recipient is zero address");
        require(amount > 0, "Executor: amount must be greater than 0");
        require(!processedCommands[commandId], "Executor: command already processed");
        require(signatures.length >= threshold, "Executor: insufficient signatures");

        // Create the message hash that validators signed
        bytes32 messageHash = keccak256(
            abi.encodePacked(commandId, recipient, amount, chainId)
        );
        bytes32 ethSignedMessageHash = messageHash.toEthSignedMessageHash();

        // Requirement 5.4: Verify signatures using ecrecover
        uint256 validSignatures = 0;
        address[] memory signers = new address[](signatures.length);

        for (uint256 i = 0; i < signatures.length; i++) {
            address signer = ethSignedMessageHash.recover(signatures[i]);

            // Check if signer is a validator
            if (!validators[signer]) {
                continue;
            }

            // Check for duplicate signers
            bool isDuplicate = false;
            for (uint256 j = 0; j < validSignatures; j++) {
                if (signers[j] == signer) {
                    isDuplicate = true;
                    break;
                }
            }

            if (!isDuplicate) {
                signers[validSignatures] = signer;
                validSignatures++;
            }
        }

        // Requirement 5.5: Reject if insufficient valid signatures
        if (validSignatures < threshold) {
            emit MintRejected(commandId, "Insufficient valid signatures");
            revert("Executor: insufficient valid signatures");
        }

        // Mark command as processed
        processedCommands[commandId] = true;

        // Requirement 1.3: Mint tokens to recipient
        token.mint(recipient, amount);

        emit MintExecuted(commandId, recipient, amount, block.timestamp);
    }

    /**
     * @dev Updates the validator set
     * Requirement 6.1: Propagate validator set changes
     * Requirement 6.4: Maintain 2/3 signature threshold
     *
     * @param newValidators Array of new validator addresses
     */
    function updateValidatorSet(address[] calldata newValidators) external onlyOwner {
        require(newValidators.length > 0, "Executor: empty validator set");

        // Remove old validators
        for (uint256 i = 0; i < validatorList.length; i++) {
            validators[validatorList[i]] = false;
            emit ValidatorRemoved(validatorList[i]);
        }

        // Clear validator list
        delete validatorList;

        // Add new validators
        for (uint256 i = 0; i < newValidators.length; i++) {
            require(newValidators[i] != address(0), "Executor: validator is zero address");
            require(!validators[newValidators[i]], "Executor: duplicate validator");

            validators[newValidators[i]] = true;
            validatorList.push(newValidators[i]);
            emit ValidatorAdded(newValidators[i]);
        }

        // Update threshold (2/3 of validators, rounded up)
        uint256 oldThreshold = threshold;
        threshold = (newValidators.length * 2 + 2) / 3; // Ceiling division for 2/3
        if (threshold == 0) {
            threshold = 1;
        }

        validatorSetVersion++;

        emit ThresholdUpdated(oldThreshold, threshold);
        emit ValidatorSetUpdated(validatorSetVersion, newValidators.length, threshold);
    }

    /**
     * @dev Adds a single validator
     * Requirement 6.2: Update contract with new validator's public key
     *
     * @param validator The validator address to add
     */
    function addValidator(address validator) external onlyOwner {
        require(validator != address(0), "Executor: validator is zero address");
        require(!validators[validator], "Executor: validator already exists");

        validators[validator] = true;
        validatorList.push(validator);

        // Recalculate threshold
        uint256 oldThreshold = threshold;
        threshold = (validatorList.length * 2 + 2) / 3;
        if (threshold == 0) {
            threshold = 1;
        }

        validatorSetVersion++;

        emit ValidatorAdded(validator);
        emit ThresholdUpdated(oldThreshold, threshold);
    }

    /**
     * @dev Removes a validator
     * Requirement 6.3: Revoke signing authority
     *
     * @param validator The validator address to remove
     */
    function removeValidator(address validator) external onlyOwner {
        require(validators[validator], "Executor: validator not found");
        require(validatorList.length > 1, "Executor: cannot remove last validator");

        validators[validator] = false;

        // Remove from list
        for (uint256 i = 0; i < validatorList.length; i++) {
            if (validatorList[i] == validator) {
                validatorList[i] = validatorList[validatorList.length - 1];
                validatorList.pop();
                break;
            }
        }

        // Recalculate threshold
        uint256 oldThreshold = threshold;
        threshold = (validatorList.length * 2 + 2) / 3;
        if (threshold == 0) {
            threshold = 1;
        }

        validatorSetVersion++;

        emit ValidatorRemoved(validator);
        emit ThresholdUpdated(oldThreshold, threshold);
    }

    /**
     * @dev Returns the current validator count
     */
    function getValidatorCount() external view returns (uint256) {
        return validatorList.length;
    }

    /**
     * @dev Returns the list of validators
     */
    function getValidators() external view returns (address[] memory) {
        return validatorList;
    }

    /**
     * @dev Checks if an address is a validator
     */
    function isValidator(address addr) external view returns (bool) {
        return validators[addr];
    }

    /**
     * @dev Checks if a command has been processed
     */
    function isCommandProcessed(bytes32 commandId) external view returns (bool) {
        return processedCommands[commandId];
    }

    /**
     * @dev Computes the message hash for a mint command (for off-chain signing)
     */
    function getMessageHash(
        bytes32 commandId,
        address recipient,
        uint256 amount
    ) external view returns (bytes32) {
        return keccak256(abi.encodePacked(commandId, recipient, amount, chainId));
    }

    /**
     * @dev Updates the token contract address
     */
    function setToken(address _token) external onlyOwner {
        require(_token != address(0), "Executor: token is zero address");
        token = BankToken(_token);
    }

    // =============================================================================
    // Error Handling and Recovery (Task 12.4)
    // =============================================================================

    // Failed command tracking for retry logic
    mapping(bytes32 => FailedCommand) public failedCommands;
    uint256 public failedCommandCount;

    struct FailedCommand {
        bytes32 commandId;
        address recipient;
        uint256 amount;
        string failureReason;
        uint256 failedAt;
        bool retried;
    }

    event CommandFailed(
        bytes32 indexed commandId,
        address indexed recipient,
        uint256 amount,
        string reason,
        uint256 timestamp
    );

    event CommandRetried(
        bytes32 indexed commandId,
        bool success
    );

    /**
     * @dev Validates a mint command without executing
     * Used for pre-flight checks and gas estimation
     * Requirement 12.4: 자동 가스 추정 및 여유분 추가
     */
    function validateMintCommand(
        bytes32 commandId,
        address recipient,
        uint256 amount,
        bytes[] calldata signatures
    ) external view returns (bool valid, string memory reason) {
        // Check if already processed
        if (processedCommands[commandId]) {
            return (false, "Command already processed");
        }

        // Check recipient
        if (recipient == address(0)) {
            return (false, "Invalid recipient address");
        }

        // Check amount
        if (amount == 0) {
            return (false, "Amount must be greater than 0");
        }

        // Check signature count
        if (signatures.length < threshold) {
            return (false, "Insufficient signatures");
        }

        // Verify signatures
        bytes32 messageHash = keccak256(
            abi.encodePacked(commandId, recipient, amount, chainId)
        );
        bytes32 ethSignedMessageHash = messageHash.toEthSignedMessageHash();

        uint256 validSignatures = 0;
        address[] memory signers = new address[](signatures.length);

        for (uint256 i = 0; i < signatures.length; i++) {
            address signer = ethSignedMessageHash.recover(signatures[i]);

            if (!validators[signer]) {
                continue;
            }

            bool isDuplicate = false;
            for (uint256 j = 0; j < validSignatures; j++) {
                if (signers[j] == signer) {
                    isDuplicate = true;
                    break;
                }
            }

            if (!isDuplicate) {
                signers[validSignatures] = signer;
                validSignatures++;
            }
        }

        if (validSignatures < threshold) {
            return (false, "Insufficient valid signatures");
        }

        return (true, "");
    }

    /**
     * @dev Returns the expected validator set version
     * Used for detecting validator set mismatch
     * Requirement 12.4: 검증자 세트 불일치 감지 및 동기화
     */
    function getValidatorSetInfo() external view returns (
        uint256 version,
        uint256 validatorCount,
        uint256 currentThreshold
    ) {
        return (validatorSetVersion, validatorList.length, threshold);
    }

    /**
     * @dev Verifies if the provided validator set matches the contract state
     * Requirement 12.4: 검증자 세트 불일치 감지
     */
    function verifyValidatorSet(
        address[] calldata expectedValidators,
        uint256 expectedVersion
    ) external view returns (bool matches, string memory reason) {
        if (expectedVersion != validatorSetVersion) {
            return (false, "Version mismatch");
        }

        if (expectedValidators.length != validatorList.length) {
            return (false, "Validator count mismatch");
        }

        for (uint256 i = 0; i < expectedValidators.length; i++) {
            if (!validators[expectedValidators[i]]) {
                return (false, "Validator not found");
            }
        }

        return (true, "");
    }

    /**
     * @dev Records a failed command for later retry
     * Internal helper for error tracking
     */
    function _recordFailedCommand(
        bytes32 commandId,
        address recipient,
        uint256 amount,
        string memory reason
    ) internal {
        failedCommands[commandId] = FailedCommand({
            commandId: commandId,
            recipient: recipient,
            amount: amount,
            failureReason: reason,
            failedAt: block.timestamp,
            retried: false
        });
        failedCommandCount++;

        emit CommandFailed(commandId, recipient, amount, reason, block.timestamp);
    }

    /**
     * @dev Gets failed command details
     */
    function getFailedCommand(bytes32 commandId) external view returns (
        address recipient,
        uint256 amount,
        string memory failureReason,
        uint256 failedAt,
        bool retried
    ) {
        FailedCommand memory cmd = failedCommands[commandId];
        return (cmd.recipient, cmd.amount, cmd.failureReason, cmd.failedAt, cmd.retried);
    }

    /**
     * @dev Estimates gas for executeMint
     * Returns estimated gas with 20% buffer
     */
    function estimateMintGas(
        bytes32 commandId,
        address recipient,
        uint256 amount,
        bytes[] calldata signatures
    ) external view returns (uint256) {
        // Base gas for state changes and token mint
        uint256 baseGas = 50000;

        // Gas per signature verification
        uint256 sigGas = signatures.length * 5000;

        // Gas for token mint
        uint256 mintGas = 30000;

        // Total with 20% buffer
        uint256 total = baseGas + sigGas + mintGas;
        return (total * 120) / 100;
    }
}
