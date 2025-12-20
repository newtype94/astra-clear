// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title BankToken
 * @dev ERC20 token representing bank-issued tokens for cross-chain transfers
 * Each bank network has its own BankToken instance
 */
contract BankToken is ERC20, Ownable {
    address public gateway;
    address public executor;

    event GatewayUpdated(address indexed oldGateway, address indexed newGateway);
    event ExecutorUpdated(address indexed oldExecutor, address indexed newExecutor);

    modifier onlyGatewayOrExecutor() {
        require(
            msg.sender == gateway || msg.sender == executor,
            "BankToken: caller is not gateway or executor"
        );
        _;
    }

    constructor(
        string memory name,
        string memory symbol,
        address initialOwner
    ) ERC20(name, symbol) Ownable(initialOwner) {}

    /**
     * @dev Sets the gateway contract address
     * @param _gateway The address of the Gateway contract
     */
    function setGateway(address _gateway) external onlyOwner {
        require(_gateway != address(0), "BankToken: gateway is zero address");
        address oldGateway = gateway;
        gateway = _gateway;
        emit GatewayUpdated(oldGateway, _gateway);
    }

    /**
     * @dev Sets the executor contract address
     * @param _executor The address of the Executor contract
     */
    function setExecutor(address _executor) external onlyOwner {
        require(_executor != address(0), "BankToken: executor is zero address");
        address oldExecutor = executor;
        executor = _executor;
        emit ExecutorUpdated(oldExecutor, _executor);
    }

    /**
     * @dev Mints tokens to an account (called by Executor after cross-chain verification)
     * @param to The recipient address
     * @param amount The amount to mint
     */
    function mint(address to, uint256 amount) external onlyGatewayOrExecutor {
        _mint(to, amount);
    }

    /**
     * @dev Burns tokens from an account (called by Gateway for cross-chain transfers)
     * @param from The address to burn from
     * @param amount The amount to burn
     */
    function burn(address from, uint256 amount) external onlyGatewayOrExecutor {
        _burn(from, amount);
    }

    /**
     * @dev Mints initial tokens to an address (for testing/setup)
     * @param to The recipient address
     * @param amount The amount to mint
     */
    function mintInitial(address to, uint256 amount) external onlyOwner {
        _mint(to, amount);
    }
}
