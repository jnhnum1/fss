all_data = [VarName2,VarName3,VarName4,VarName5];

clear final_data
count = ones(5,1);
for ii = 1:size(all_data,1)
    for jj = 0:4
        if(jj == all_data(ii,1))
            final_data(count(jj+1),:,jj+1) = all_data(ii,:);
            count(jj+1) = count(jj+1) + 1;
        end
    end
end

new_data = [final_data(:,2,1),sum(final_data(:,4,:),3)];