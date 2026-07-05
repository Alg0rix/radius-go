package radiusctl

import (
	"fmt"

	"github.com/spf13/cobra"
)

// voucherCmd returns the `radiusctl voucher` subcommand group.
func voucherCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "voucher",
		Short: "Manage voucher packages and vouchers",
	}
	cmd.AddCommand(
		voucherListCmd(),
		voucherGenerateCmd(),
		voucherBalanceCmd(),
		voucherPackageCmd(),
	)
	return cmd
}

func voucherListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all vouchers",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := client.ListVouchers(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Println("no vouchers")
				return nil
			}
			w := NewWriter()
			w.Row("ID", "USERNAME", "PACKAGE", "ENABLED", "EXPIRES")
			for _, v := range items {
				expires := "-"
				if v.ExpiresAt != nil {
					expires = v.ExpiresAt.String()
				}
				w.Row(v.ID, v.Username, v.VoucherPackageID, yesNo(v.Enabled), expires)
			}
			w.Flush()
			return nil
		},
	}
}

func voucherGenerateCmd() *cobra.Command {
	var (
		packageID, codeFormat, passwordMode, customCode, customPassword string
		count, codeLength                                               int
	)
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate voucher(s) from a package",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := GenerateVoucherRequest{
				PackageID:      packageID,
				Count:          count,
				CodeFormat:     codeFormat,
				CodeLength:     codeLength,
				CustomCode:     customCode,
				PasswordMode:   passwordMode,
				CustomPassword: customPassword,
			}
			items, err := client.GenerateVouchers(cmd.Context(), req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Println("no vouchers generated")
				return nil
			}
			w := NewWriter()
			w.Row("USERNAME", "PASSWORD", "PACKAGE")
			for _, v := range items {
				w.Row(v.Username, v.Password, v.VoucherPackageID)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVarP(&packageID, "package-id", "p", "", "Voucher package UUID (required)")
	cmd.Flags().IntVarP(&count, "count", "c", 1, "Number of vouchers to generate")
	cmd.Flags().StringVar(&codeFormat, "code-format", "random", "Code format: random|custom")
	cmd.Flags().IntVar(&codeLength, "code-length", 8, "Random code length")
	cmd.Flags().StringVar(&customCode, "custom-code", "", "Custom code (code-format=custom)")
	cmd.Flags().StringVar(&passwordMode, "password-mode", "same_as_user", "Password mode: same_as_user|random|custom")
	cmd.Flags().StringVar(&customPassword, "custom-password", "", "Custom password (password-mode=custom)")
	cmd.MarkFlagRequired("package-id")
	return cmd
}

func voucherBalanceCmd() *cobra.Command {
	var code string
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Show remaining balance for a voucher code",
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := client.VoucherBalance(cmd.Context(), code)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(b)
				return nil
			}
			KV(
				[2]string{"username", b.Username},
				[2]string{"package", b.PackageName},
				[2]string{"enabled", yesNo(b.Enabled)},
				[2]string{"time_limit_type", strOr(b.TimeLimitType, "-")},
				[2]string{"time_limit_seconds", itoa(b.TimeLimitSeconds)},
				[2]string{"usage_seconds_used", itoa(b.UsageSecondsUsed)},
				[2]string{"usage_seconds_remaining", itoa(b.UsageSecondsRemaining)},
				[2]string{"data_cap_bytes", humanBytes(b.DataCapBytes)},
				[2]string{"data_bytes_used", humanBytes(b.DataBytesUsed)},
				[2]string{"data_bytes_remaining", humanBytes(b.DataBytesRemaining)},
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&code, "code", "c", "", "Voucher code / username (required)")
	cmd.MarkFlagRequired("code")
	return cmd
}

// --- voucher package management ---

func voucherPackageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Manage voucher packages",
	}
	cmd.AddCommand(
		voucherPackageListCmd(),
		voucherPackageCreateCmd(),
		voucherPackageUpdateCmd(),
		voucherPackageDeleteCmd(),
	)
	return cmd
}

func voucherPackageListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all voucher packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := client.ListVoucherPackages(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Println("no voucher packages")
				return nil
			}
			w := NewWriter()
			w.Row("ID", "NAME", "PRICE", "SPEED UP/DOWN", "TIME LIMIT", "ENABLED")
			for _, p := range items {
				w.Row(
					p.ID,
					p.Name,
					fmt.Sprintf("%.2f", p.Price),
					fmt.Sprintf("%d/%d kbps", p.SpeedUploadKbps, p.SpeedDownloadKbps),
					fmt.Sprintf("%s/%ds", p.TimeLimitType, p.TimeLimitSeconds),
					yesNo(p.Enabled),
				)
			}
			w.Flush()
			return nil
		},
	}
}

func voucherPackageCreateCmd() *cobra.Command {
	var (
		name, desc, timeLimitType       string
		price                           float64
		speedUp, speedDown              int
		dataCapBytes                    int64
		timeLimitSeconds                int
		maxConcurrent                   int
		addressPool, primaryDNS, secondaryDNS string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a voucher package",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := CreateVoucherPackageRequest{
				Name:               name,
				Description:        desc,
				Price:              price,
				SpeedUploadKbps:    speedUp,
				SpeedDownloadKbps:  speedDown,
				DataCapBytes:       dataCapBytes,
				TimeLimitType:      timeLimitType,
				TimeLimitSeconds:   timeLimitSeconds,
				MaxConcurrentUsers: maxConcurrent,
				AddressPool:        addressPool,
				PrimaryDNS:         primaryDNS,
				SecondaryDNS:       secondaryDNS,
			}
			p, err := client.CreateVoucherPackage(cmd.Context(), req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(p)
				return nil
			}
			printVoucherPackage(p)
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "Package name (required)")
	cmd.Flags().StringVarP(&desc, "description", "d", "", "Description")
	cmd.Flags().Float64VarP(&price, "price", "p", 0, "Price")
	cmd.Flags().IntVar(&speedUp, "speed-up", 0, "Upload speed (kbps)")
	cmd.Flags().IntVar(&speedDown, "speed-down", 0, "Download speed (kbps)")
	cmd.Flags().Int64Var(&dataCapBytes, "data-cap-bytes", 0, "Data cap (bytes)")
	cmd.Flags().StringVar(&timeLimitType, "time-limit-type", "usage", "Time limit type: calendar|usage")
	cmd.Flags().IntVar(&timeLimitSeconds, "time-limit-seconds", 0, "Time limit (seconds)")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 0, "Max concurrent users")
	cmd.Flags().StringVar(&addressPool, "address-pool", "", "Hotspot address pool name")
	cmd.Flags().StringVar(&primaryDNS, "primary-dns", "", "Primary DNS server")
	cmd.Flags().StringVar(&secondaryDNS, "secondary-dns", "", "Secondary DNS server")
	cmd.MarkFlagRequired("name")
	return cmd
}

func voucherPackageUpdateCmd() *cobra.Command {
	var (
		id, name, desc, timeLimitType           string
		price                                   float64
		speedUp, speedDown                      int
		dataCapBytes                            int64
		timeLimitSeconds                        int
		maxConcurrent                           int
		addressPool, primaryDNS, secondaryDNS   string
		enabled, disabled                       bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a voucher package",
		RunE: func(cmd *cobra.Command, args []string) error {
			if enabled && disabled {
				return fmt.Errorf("--enable and --disable are mutually exclusive")
			}
			req := UpdateVoucherPackageRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("description") {
				req.Description = &desc
			}
			if cmd.Flags().Changed("price") {
				req.Price = &price
			}
			if cmd.Flags().Changed("speed-up") {
				req.SpeedUploadKbps = &speedUp
			}
			if cmd.Flags().Changed("speed-down") {
				req.SpeedDownloadKbps = &speedDown
			}
			if cmd.Flags().Changed("data-cap-bytes") {
				req.DataCapBytes = &dataCapBytes
			}
			if cmd.Flags().Changed("time-limit-type") {
				req.TimeLimitType = &timeLimitType
			}
			if cmd.Flags().Changed("time-limit-seconds") {
				req.TimeLimitSeconds = &timeLimitSeconds
			}
			if cmd.Flags().Changed("max-concurrent") {
				req.MaxConcurrentUsers = &maxConcurrent
			}
			if cmd.Flags().Changed("address-pool") {
				req.AddressPool = &addressPool
			}
			if cmd.Flags().Changed("primary-dns") {
				req.PrimaryDNS = &primaryDNS
			}
			if cmd.Flags().Changed("secondary-dns") {
				req.SecondaryDNS = &secondaryDNS
			}
			if enabled || disabled {
				v := enabled
				req.Enabled = &v
			}
			p, err := client.UpdateVoucherPackage(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(p)
				return nil
			}
			printVoucherPackage(p)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "Package UUID (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Package name")
	cmd.Flags().StringVarP(&desc, "description", "d", "", "Description")
	cmd.Flags().Float64VarP(&price, "price", "p", 0, "Price")
	cmd.Flags().IntVar(&speedUp, "speed-up", 0, "Upload speed (kbps)")
	cmd.Flags().IntVar(&speedDown, "speed-down", 0, "Download speed (kbps)")
	cmd.Flags().Int64Var(&dataCapBytes, "data-cap-bytes", 0, "Data cap (bytes)")
	cmd.Flags().StringVar(&timeLimitType, "time-limit-type", "", "Time limit type: calendar|usage")
	cmd.Flags().IntVar(&timeLimitSeconds, "time-limit-seconds", 0, "Time limit (seconds)")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 0, "Max concurrent users")
	cmd.Flags().StringVar(&addressPool, "address-pool", "", "Hotspot address pool name")
	cmd.Flags().StringVar(&primaryDNS, "primary-dns", "", "Primary DNS server")
	cmd.Flags().StringVar(&secondaryDNS, "secondary-dns", "", "Secondary DNS server")
	cmd.Flags().BoolVar(&enabled, "enable", false, "Enable the package")
	cmd.Flags().BoolVar(&disabled, "disable", false, "Disable the package")
	cmd.MarkFlagRequired("id")
	return cmd
}

func voucherPackageDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a voucher package",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := client.DeleteVoucherPackage(cmd.Context(), id)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(r)
				return nil
			}
			fmt.Printf("voucher package %s deleted\n", r.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "Package UUID (required)")
	cmd.MarkFlagRequired("id")
	return cmd
}

// printVoucherPackage renders a single voucher package as key-value pairs.
func printVoucherPackage(p VoucherPackage) {
	KV(
		[2]string{"id", p.ID},
		[2]string{"name", p.Name},
		[2]string{"description", strOr(p.Description, "-")},
		[2]string{"price", fmt.Sprintf("%.2f", p.Price)},
		[2]string{"speed_upload_kbps", itoa(p.SpeedUploadKbps)},
		[2]string{"speed_download_kbps", itoa(p.SpeedDownloadKbps)},
		[2]string{"data_cap_bytes", humanBytes(p.DataCapBytes)},
		[2]string{"time_limit_type", p.TimeLimitType},
		[2]string{"time_limit_seconds", itoa(p.TimeLimitSeconds)},
		[2]string{"max_concurrent_users", itoa(p.MaxConcurrentUsers)},
		[2]string{"address_pool", strOr(p.AddressPool, "-")},
		[2]string{"primary_dns", strOr(p.PrimaryDNS, "-")},
		[2]string{"secondary_dns", strOr(p.SecondaryDNS, "-")},
		[2]string{"enabled", yesNo(p.Enabled)},
	)
}
