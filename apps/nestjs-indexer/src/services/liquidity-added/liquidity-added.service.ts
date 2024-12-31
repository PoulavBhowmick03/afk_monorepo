import { Injectable, Logger } from '@nestjs/common';
import { PrismaService } from 'src/prisma/prisma.service';
import { BuyToken } from './interfaces';

@Injectable()
export class LiquidityAddedService {
  private readonly logger = new Logger(LiquidityAddedService.name);
  constructor(private readonly prismaService: PrismaService) {}

  async create(data: BuyToken) {
    try {
      const buyTokenRecord =
        await this.prismaService.token_transactions.findUnique({
          where: { transfer_id: data.transferId },
        });

      if (buyTokenRecord) {
        this.logger.warn(
          `Record with transfer ID ${data.transferId} already exists`,
        );
        return;
      }

      const tokenLaunchRecord = await this.prismaService.token_launch.findFirst(
        { where: { memecoin_address: data.memecoinAddress } },
      );

      if (!tokenLaunchRecord) {
        this.logger.warn(
          `Record with memecoin address ${data.memecoinAddress} doesn't exists`,
        );
      } else {

        await this.prismaService.token_launch.update({
          where: { transaction_hash: tokenLaunchRecord.transaction_hash },
          data: {
            is_liquidity_added: true,
          },
        });
      }

    } catch (error) {
      this.logger.error(
        `Error creating buy token record: ${error.message}`,
        error.stack,
      );
    }
  }
}