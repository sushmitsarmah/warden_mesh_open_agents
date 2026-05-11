//! # VulnerableVault — Deliberately Insecure Solana Program
//!
//! DO NOT DEPLOY. This program contains intentional security vulnerabilities
//! for testing the autonomous security swarm's Solana analyzer pipeline.
//!
//! ## Vulnerabilities (one per instruction)
//!
//! | # | Instruction       | Vulnerability Class                        | Severity |
//! |---|-------------------|--------------------------------------------|----------|
//! | 1 | `deposit`         | Integer overflow (unchecked arithmetic)    | High     |
//! | 2 | `withdraw`        | Missing signer check on authority          | Critical |
//! | 3 | `set_fee`         | Missing timelock / no access control       | High     |
//! | 4 | `cpi_proxy`       | Arbitrary CPI (attacker-controlled target) | Critical |
//! | 5 | `load_user`       | Missing ownership check on data account    | High     |
//! | 6 | `create_escrow`   | Missing bump canonicalization              | High     |
//! | 7 | `close_and_send`  | Insecure account close (lamport leak)      | Medium   |
//! | 8 | `admin_withdraw`  | Account confusion (unchecked deserialize)  | Critical |

use anchor_lang::prelude::*;
use anchor_lang::solana_program::program::invoke;
use anchor_lang::solana_program::instruction::Instruction;
use anchor_spl::token::{self, Token, TokenAccount, Transfer};

declare_id!("VuLnVauLt111111111111111111111111111111111");

// ─────────────────────────────────────────────────────────────
// Program State
// ─────────────────────────────────────────────────────────────

#[account]
pub struct Vault {
    pub authority: Pubkey,
    pub total_deposits: u64,
    pub fee_bps: u16,         // basis points
    pub bump: u8,
}

#[account]
pub struct UserAccount {
    pub owner: Pubkey,
    pub deposited: u64,
    pub bump: u8,
}

// A fake "admin" account — same layout as UserAccount but used for different purpose.
// The admin_withdraw instruction confuses these two types.
#[account]
pub struct AdminAccount {
    pub owner: Pubkey,
    pub privilege_level: u64,  // overlaps with `deposited` field in UserAccount
    pub bump: u8,
}

// ─────────────────────────────────────────────────────────────
// Program Entry Points
// ─────────────────────────────────────────────────────────────

#[program]
pub mod vulnerable_vault {
    use super::*;

    /// Initialize the vault.
    pub fn initialize(ctx: Context<Initialize>, fee_bps: u16) -> Result<()> {
        let vault = &mut ctx.accounts.vault;
        vault.authority = ctx.accounts.authority.key();
        vault.total_deposits = 0;
        vault.fee_bps = fee_bps;
        vault.bump = ctx.bumps.vault;
        Ok(())
    }

    /// VULN 1 — Integer Overflow
    ///
    /// Uses `+` instead of `checked_add`. On overflow the deposit count wraps
    /// to zero, letting an attacker donate dust to reset the vault's accounting.
    pub fn deposit(ctx: Context<Deposit>, amount: u64) -> Result<()> {
        let vault = &mut ctx.accounts.vault;
        let user = &mut ctx.accounts.user_account;

        // BUG: unchecked arithmetic — wraps on overflow
        vault.total_deposits = vault.total_deposits + amount;
        user.deposited = user.deposited + amount;

        // Transfer SOL from user to vault PDA
        let ix = anchor_lang::solana_program::system_instruction::transfer(
            &ctx.accounts.depositor.key(),
            &ctx.accounts.vault.key(),
            amount,
        );
        anchor_lang::solana_program::program::invoke(
            &ix,
            &[
                ctx.accounts.depositor.to_account_info(),
                ctx.accounts.vault.to_account_info(),
            ],
        )?;

        Ok(())
    }

    /// VULN 2 — Missing Signer Check
    ///
    /// The `authority` account is passed in but never verified with `is_signer`.
    /// Anyone can call this and drain the vault by supplying any `authority` pubkey.
    pub fn withdraw(ctx: Context<Withdraw>, amount: u64) -> Result<()> {
        // BUG: authority.is_signer is never checked — Anchor's `Signer<>` type
        // is NOT used here; `AccountInfo` is used directly.
        // An attacker passes authority = vault.authority (matching pubkey)
        // but does not need to sign with that key.

        let vault = &ctx.accounts.vault;
        let vault_lamports = vault.to_account_info().lamports();
        require!(vault_lamports >= amount, VaultError::InsufficientFunds);

        // Transfer out of vault PDA
        **ctx.accounts.vault.to_account_info().try_borrow_mut_lamports()? -= amount;
        **ctx.accounts.recipient.to_account_info().try_borrow_mut_lamports()? += amount;

        Ok(())
    }

    /// VULN 3 — No Access Control on Admin Function
    ///
    /// Any caller can change the fee. There is no `onlyOwner` equivalent,
    /// no timelock, and no multisig requirement.
    pub fn set_fee(ctx: Context<SetFee>, new_fee_bps: u16) -> Result<()> {
        // BUG: no check that ctx.accounts.caller == vault.authority
        ctx.accounts.vault.fee_bps = new_fee_bps;
        Ok(())
    }

    /// VULN 4 — Arbitrary CPI
    ///
    /// The target program is taken from an account field — the caller can supply
    /// any program ID and the vault will invoke it with the vault's signing authority.
    pub fn cpi_proxy(ctx: Context<CpiProxy>, instruction_data: Vec<u8>) -> Result<()> {
        // BUG: `target_program` comes from an attacker-supplied account, not a
        // hardcoded trusted constant. The vault PDA signs the CPI, giving the
        // attacker's program authority over vault-owned accounts.
        let target_program = ctx.accounts.target_program.key();

        let ix = Instruction {
            program_id: target_program,
            accounts: vec![],
            data: instruction_data,
        };

        // Vault PDA signs the cross-program call
        let vault_seeds = &[
            b"vault",
            ctx.accounts.vault.authority.as_ref(),
            &[ctx.accounts.vault.bump],
        ];
        let signer_seeds = &[&vault_seeds[..]];

        anchor_lang::solana_program::program::invoke_signed(
            &ix,
            &[ctx.accounts.vault.to_account_info()],
            signer_seeds,
        )?;

        Ok(())
    }

    /// VULN 5 — Missing Ownership Check
    ///
    /// Reads arbitrary account data without verifying `account.owner == program_id`.
    /// An attacker can pass a crafted account owned by a different program, causing
    /// the vault to act on incorrect user balances.
    pub fn load_user(ctx: Context<LoadUser>) -> Result<()> {
        // BUG: no account.owner check — any account with the right byte layout passes
        let data = ctx.accounts.user_data.data.borrow();
        // Directly casting raw bytes without discriminator verification
        let deposited = u64::from_le_bytes(data[8..16].try_into().unwrap());
        msg!("User deposited: {}", deposited);
        Ok(())
    }

    /// VULN 6 — Missing Bump Seed Canonicalization
    ///
    /// Uses `create_program_address` with a caller-supplied bump instead of
    /// `find_program_address`. An attacker can grind non-canonical bumps to
    /// derive alternative PDAs that pass the address check.
    pub fn create_escrow(ctx: Context<CreateEscrow>, user_bump: u8) -> Result<()> {
        // BUG: caller controls `user_bump` — should use canonical bump from
        // `find_program_address`, not an arbitrary caller-supplied value.
        let seeds = &[b"escrow", ctx.accounts.user.key.as_ref(), &[user_bump]];
        let derived = Pubkey::create_program_address(seeds, ctx.program_id)
            .map_err(|_| VaultError::InvalidBump)?;

        require_keys_eq!(derived, ctx.accounts.escrow.key(), VaultError::InvalidBump);

        msg!("Escrow created at {}", derived);
        Ok(())
    }

    /// VULN 7 — Insecure Account Close (Lamport Leak)
    ///
    /// Closes a user account but does NOT zero the data and does NOT use
    /// Anchor's `close` constraint. An attacker can "revive" the account
    /// in the same transaction by re-funding it before the runtime reclaims it.
    pub fn close_and_send(ctx: Context<CloseAndSend>) -> Result<()> {
        let user_account = &mut ctx.accounts.user_account;
        let destination = &ctx.accounts.destination;

        // BUG: just moves lamports — does not zero account data, does not
        // set lamports to 0 atomically, and does not use the `close` attribute.
        let lamports = user_account.to_account_info().lamports();
        **user_account.to_account_info().try_borrow_mut_lamports()? -= lamports;
        **destination.to_account_info().try_borrow_mut_lamports()? += lamports;

        // Data is NOT zeroed — account can be revived in the same transaction
        Ok(())
    }

    /// VULN 8 — Account Confusion via Unchecked Deserialize
    ///
    /// Uses `try_deserialize_unchecked` which skips the 8-byte discriminator check.
    /// An attacker supplies a `UserAccount` where an `AdminAccount` is expected,
    /// and the overlapping `deposited` field is interpreted as `privilege_level`,
    /// granting false admin access.
    pub fn admin_withdraw(ctx: Context<AdminWithdraw>, amount: u64) -> Result<()> {
        let raw_data = ctx.accounts.admin_account.data.borrow();

        // BUG: unchecked deserialization — skips discriminator check.
        // UserAccount { owner, deposited: 999 } is accepted as
        // AdminAccount { owner, privilege_level: 999 } because the byte layout
        // is identical and the discriminator (first 8 bytes) is not verified.
        let admin = AdminAccount::try_deserialize_unchecked(&mut raw_data.as_ref())
            .map_err(|_| VaultError::InvalidAccount)?;

        // Any account with privilege_level > 0 passes — attacker sets deposited = 1
        require!(admin.privilege_level > 0, VaultError::NotAdmin);

        **ctx.accounts.vault.to_account_info().try_borrow_mut_lamports()? -= amount;
        **ctx.accounts.attacker.to_account_info().try_borrow_mut_lamports()? += amount;

        Ok(())
    }
}

// ─────────────────────────────────────────────────────────────
// Account Contexts
// ─────────────────────────────────────────────────────────────

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(
        init,
        payer = authority,
        space = 8 + 32 + 8 + 2 + 1,
        seeds = [b"vault", authority.key().as_ref()],
        bump
    )]
    pub vault: Account<'info, Vault>,
    #[account(mut)]
    pub authority: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct Deposit<'info> {
    #[account(mut, seeds = [b"vault", vault.authority.as_ref()], bump = vault.bump)]
    pub vault: Account<'info, Vault>,
    #[account(
        init_if_needed,
        payer = depositor,
        space = 8 + 32 + 8 + 1,
        seeds = [b"user", depositor.key().as_ref()],
        bump
    )]
    pub user_account: Account<'info, UserAccount>,
    #[account(mut)]
    pub depositor: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct Withdraw<'info> {
    #[account(mut, seeds = [b"vault", vault.authority.as_ref()], bump = vault.bump)]
    pub vault: Account<'info, Vault>,
    // BUG: AccountInfo instead of Signer — is_signer never enforced
    pub authority: AccountInfo<'info>,
    #[account(mut)]
    pub recipient: AccountInfo<'info>,
}

#[derive(Accounts)]
pub struct SetFee<'info> {
    #[account(mut)]
    pub vault: Account<'info, Vault>,
    // BUG: caller is not required to be the vault authority
    pub caller: AccountInfo<'info>,
}

#[derive(Accounts)]
pub struct CpiProxy<'info> {
    #[account(mut, seeds = [b"vault", vault.authority.as_ref()], bump = vault.bump)]
    pub vault: Account<'info, Vault>,
    // BUG: target_program is attacker-controlled — not validated against a trusted constant
    pub target_program: AccountInfo<'info>,
    pub caller: Signer<'info>,
}

#[derive(Accounts)]
pub struct LoadUser<'info> {
    pub vault: Account<'info, Vault>,
    // BUG: AccountInfo with no owner check — accepts any account
    pub user_data: AccountInfo<'info>,
}

#[derive(Accounts)]
pub struct CreateEscrow<'info> {
    #[account(mut)]
    pub user: Signer<'info>,
    // BUG: escrow is not derived with find_program_address — caller supplies bump
    #[account(mut)]
    pub escrow: AccountInfo<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct CloseAndSend<'info> {
    #[account(mut)]
    pub user_account: Account<'info, UserAccount>,
    #[account(mut)]
    pub destination: AccountInfo<'info>,
    pub user: Signer<'info>,
}

#[derive(Accounts)]
pub struct AdminWithdraw<'info> {
    #[account(mut, seeds = [b"vault", vault.authority.as_ref()], bump = vault.bump)]
    pub vault: Account<'info, Vault>,
    // BUG: raw AccountInfo — discriminator not checked by Anchor
    pub admin_account: AccountInfo<'info>,
    #[account(mut)]
    pub attacker: AccountInfo<'info>,
}

// ─────────────────────────────────────────────────────────────
// Errors
// ─────────────────────────────────────────────────────────────

#[error_code]
pub enum VaultError {
    #[msg("Insufficient funds in vault")]
    InsufficientFunds,
    #[msg("Invalid bump seed")]
    InvalidBump,
    #[msg("Not an admin account")]
    NotAdmin,
    #[msg("Invalid account supplied")]
    InvalidAccount,
}
