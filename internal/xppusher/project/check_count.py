import asyncio
import yaml
from pixiv_client import PixivClient

async def main():
    with open('config.yaml', 'r', encoding='utf-8') as f:
        config = yaml.safe_load(f)
        
    client = PixivClient(refresh_token=config['pixiv']['refresh_token'])
    await client.login()
    
    user_id = config['pixiv']['user_id']
    detail = await client.api.user_detail(user_id)
    print(f"User ID: {user_id}")
    print(f"Total Public Bookmarks: {detail.profile.total_illust_bookmarks_public}")
    
    await client.close()

if __name__ == "__main__":
    asyncio.run(main())
