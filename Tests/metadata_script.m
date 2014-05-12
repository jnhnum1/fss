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

new_data = [sum(final_data(:,4,:),3),final_data(:,2,1)];
new_data2 = [sum(final_data(:,4,:),3)-sum(final_data(:,3,:),3),final_data(:,2,1)];

figure; plot(new_data(:,1),new_data(:,2)); title('Meta-Data Overhead'); xlabel('Files Created'); ylabel(' Meta-Data Overhead (Bytes)')

figure; plot(new_data2(:,1),new_data2(:,2));